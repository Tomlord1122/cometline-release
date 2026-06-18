package agent

import (
	"context"
	"fmt"
	"strings"
	"time"

	cometsdk "github.com/cometline/comet-sdk"
	"github.com/cometline/comet-sdk/llm"
	"github.com/cometline/cometmind/internal/event"
	"github.com/cometline/cometmind/internal/logging"
	"github.com/cometline/cometmind/internal/memory"
	"github.com/cometline/cometmind/internal/session"
	"github.com/cometline/cometmind/internal/tools"
)

// TurnStore is the narrow persistence seam the agent loop drives. It is the
// subset of session.Service the Runner actually needs, declared here on the
// consumer side so the loop can be unit-tested with an in-memory fake instead
// of a live SQLite database. *session.Service satisfies it.
type TurnStore interface {
	BuildSDKMessages(ctx context.Context, sessionID string) ([]cometsdk.Message, error)
	SaveTokenUsage(ctx context.Context, sessionID string, u cometsdk.TokenUsage) error
	AppendAssistantStep(ctx context.Context, sessionID, text string, reasoningBlocks []cometsdk.Block, toolCalls []cometsdk.ToolCallBlock) (session.Message, map[string]string, error)
	UpdateToolCallResult(ctx context.Context, toolCallID, result string, durMs int64, exit *int64) error
	AppendToolResultMessage(ctx context.Context, sessionID, toolCallID, output string, isErr bool) (session.Message, error)
}

// Runner executes the persisted agent loop for one user turn (which may span many tool steps).
type Runner struct {
	Provider cometsdk.Provider
	Sessions TurnStore
	Memory   *memory.Service
	Registry *tools.Registry

	MaxSteps     int
	MaxTokens    int
	SystemPrompt string
	SkillIndex   string

	// MemorySem is an optional semaphore that bounds the number of
	// extractMemoryBackground goroutines that may run concurrently across all
	// sessions. When non-nil, each background goroutine acquires one slot
	// before starting and releases it on completion. A nil value means
	// unlimited (the previous behaviour).
	MemorySem chan struct{}
}

// Run streams CometMind-native events on ch until the turn completes or ctx is cancelled.
// The caller must receive until the channel closes.
func (r *Runner) Run(ctx context.Context, turn session.AgentTurn, ch chan<- event.Event) error {
	doneSent := false
	sendDone := func() {
		if doneSent {
			return
		}
		ch <- event.Done()
		doneSent = true
	}
	defer sendDone()

	completeTurn := func() error {
		sendDone()
		go r.extractMemoryBackground(context.WithoutCancel(ctx), turn)
		return nil
	}

	if r.MaxSteps <= 0 {
		r.MaxSteps = 50
	}
	if r.MaxTokens <= 0 {
		r.MaxTokens = 2048
	}

	steps := 0
	for steps < r.MaxSteps {
		msgs, err := r.Sessions.BuildSDKMessages(ctx, turn.ID)
		if err != nil {
			ch <- event.Errorf(err.Error(), "history")
			return err
		}
		logging.L().Info("agent.step.start", "session", turn.ID, "step", steps+1, "model", turn.ModelID, "messages", len(msgs), "max_tokens", r.MaxTokens)

		system := r.systemPrompt()
		if r.Memory != nil && r.Memory.Enabled() && steps == 0 {
			query := memory.BuildRetrievalQuery(memory.RetrievalQueryInput{
				Messages: msgs,
			})
			mems, memErr := r.Memory.RetrieveForTurn(ctx, query)
			if memErr != nil {
				logging.L().Error("memory.retrieve.failed", "session", turn.ID, "error", memErr)
				ch <- event.Errorf(memErr.Error(), "memory")
			} else if len(mems) > 0 {
				logging.L().Info("memory.injected", "session", turn.ID, "count", len(mems))
				system += memory.FormatForPrompt(mems)
				wire := make([]event.MemoryWire, len(mems))
				for i, m := range mems {
					wire[i] = event.MemoryWire{
						ID:              m.ID,
						Content:         m.Content,
						Kind:            m.Kind,
						Similarity:      m.Similarity,
						EffectiveWeight: m.EffectiveWeight,
					}
				}
				ch <- event.MemoryInjected(wire)
			}
		}

		req := BuildRequest(turn.ModelID, system, msgs, r.Registry.CometSDK(), r.MaxTokens)
		stream := llm.StreamMessage(ctx, r.Provider, req)

		for ev := range stream.Events() {
			switch e := ev.(type) {
			case cometsdk.TextDeltaEvent:
				ch <- event.TextDelta(e.Text)
			case cometsdk.ReasoningStartEvent:
				ch <- event.ReasoningStart()
			case cometsdk.ReasoningContentEvent:
				ch <- event.ReasoningDelta(e.Text)
			case cometsdk.ToolCallDoneEvent:
				ch <- event.ToolCall(e.ID, e.Name, []byte(e.Input))
			case cometsdk.StepFinishEvent:
				ch <- event.StepFinish(e.Usage)
			}
		}

		result, err := stream.Result()
		if err != nil {
			logging.L().Error("agent.step.failed", "session", turn.ID, "step", steps+1, "error", err)
			ch <- event.Errorf(err.Error(), "llm")
			return err
		}
		logging.L().Info("agent.step.finish", "session", turn.ID, "step", steps+1, "finish_reason", string(result.FinishReason), "tool_calls", len(result.ToolCalls), "input_tokens", result.Usage.InputTokens, "output_tokens", result.Usage.OutputTokens)

		if err := r.Sessions.SaveTokenUsage(ctx, turn.ID, result.Usage); err != nil {
			ch <- event.Errorf(err.Error(), "db")
			return err
		}

		text := assistantPlainText(result.Message)
		reasoningBlocks := result.Message.ReasoningContent
		_, persistedToolIDs, err := r.Sessions.AppendAssistantStep(ctx, turn.ID, text, reasoningBlocks, result.ToolCalls)
		if err != nil {
			ch <- event.Errorf(err.Error(), "db")
			return err
		}

		switch result.FinishReason {
		case cometsdk.FinishStop, cometsdk.FinishMaxTokens:
			return completeTurn()
		}
		if len(result.ToolCalls) == 0 {
			return completeTurn()
		}

		for _, tc := range result.ToolCalls {
			persistedID := persistedToolIDs[tc.ID]
			if persistedID == "" {
				ch <- event.Errorf("missing persisted tool call id", "db")
				return fmt.Errorf("missing persisted tool call id for %s", tc.ID)
			}
			start := time.Now()
			logging.L().Info("tool.call.start", "session", turn.ID, "tool", tc.Name, "tool_call_id", tc.ID, "input_bytes", len(tc.Input))
			toolCtx := tools.WithToolSession(ctx, turn.ID)
			toolCtx = tools.WithProgress(toolCtx, func(ev event.Event) {
				ch <- ev
			})
			res, execErr := r.Registry.Execute(toolCtx, tc.Name, tc.Input)
			dur := time.Since(start).Milliseconds()
			logging.L().Info("tool.call.finish", "session", turn.ID, "tool", tc.Name, "tool_call_id", tc.ID, "ok", res.OK && execErr == nil, "duration_ms", dur, "output_bytes", len(res.Output))

			out := res.Output
			isErr := !res.OK
			if execErr != nil {
				isErr = true
				out = fmt.Sprintf("%s\n(execute error: %v)", out, execErr)
			}

			exit := int64PtrFromIntPtr(res.ExitCode)
			if err := r.Sessions.UpdateToolCallResult(ctx, persistedID, out, dur, exit); err != nil {
				ch <- event.Errorf(err.Error(), "db")
				return err
			}
			if _, err := r.Sessions.AppendToolResultMessage(ctx, turn.ID, persistedID, out, isErr); err != nil {
				ch <- event.Errorf(err.Error(), "db")
				return err
			}

			toolErr := ""
			if isErr {
				toolErr = out
			}
			ch <- event.ToolResult(tc.ID, tc.Name, out, toolErr)
		}

		steps++
	}

	ch <- event.Errorf("max steps exceeded", "max_steps")
	return fmt.Errorf("max steps exceeded")
}

func (r *Runner) extractMemoryBackground(ctx context.Context, turn session.AgentTurn) {
	if r.Memory == nil || !r.Memory.Enabled() {
		return
	}
	// Honour the optional concurrency cap: acquire a slot before doing any
	// work and release it when done. This prevents N simultaneous session
	// completions from spawning N unbounded LLM API calls and SQLite writes.
	if r.MemorySem != nil {
		select {
		case r.MemorySem <- struct{}{}:
			defer func() { <-r.MemorySem }()
		case <-ctx.Done():
			return
		}
	}
	// Best-effort persistence only; the turn SSE stream has already closed.
	_, _ = r.Memory.ExtractAfterTurn(ctx, turn.ID, turn.ModelID, r.Provider)
}

func (r *Runner) systemPrompt() string {
	base := strings.TrimSpace(r.SystemPrompt)
	if base == "" {
		base = DefaultSystemPrompt
	}
	if strings.TrimSpace(r.SkillIndex) == "" {
		return base
	}
	return base + r.SkillIndex
}

func int64PtrFromIntPtr(v *int) *int64 {
	if v == nil {
		return nil
	}
	x := int64(*v)
	return &x
}

func assistantPlainText(m cometsdk.Message) string {
	var b strings.Builder
	for _, bl := range m.Content {
		if tb, ok := bl.(cometsdk.TextBlock); ok {
			b.WriteString(tb.Text)
		}
	}
	return b.String()
}
