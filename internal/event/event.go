package event

import (
	"encoding/json"

	cometsdk "github.com/cometline/comet-sdk"
)

// Kind identifies a CometMind-native runtime event. The same value is the SSE
// "type" discriminator on the wire, so adding a Kind is a wire-contract change.
type Kind string

const (
	KindTextDelta      Kind = "text_delta"
	KindReasoningStart Kind = "reasoning_start"
	KindReasoningDelta Kind = "reasoning_delta"
	KindToolCall       Kind = "tool_call"
	KindToolResult     Kind = "tool_result"
	KindStepFinish     Kind = "step_finish"
	KindError          Kind = "error"
	KindDone           Kind = "done"
)

// Usage mirrors the SSE token-usage payload (one source of truth for the wire).
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheRead    int `json:"cache_read"`
	CacheWrite   int `json:"cache_write"`
}

// Event is the single runtime event type shared by the agent runner, the SSE
// server, and the CLI. It is also the SSE wire shape: MarshalJSON emits
// exactly the fields each Kind carries, discriminated by "type". Field names are
// per-Kind by contract — reasoning_delta carries "text", text_delta carries
// "delta" — so consumers read the field that matches the Kind.
type Event struct {
	Kind Kind

	// text_delta
	Delta string
	// reasoning_delta
	Text string
	// tool_call / tool_result
	ID      string
	Tool    string
	Input   []byte // tool_call: JSON object bytes; empty marshals as {}
	Output  string // tool_result
	ToolErr string // tool_result: empty if success
	// step_finish
	Usage Usage
	// error
	Message string
	Code    string
}

// MarshalJSON renders the SSE wire frame body for this event. The output is the
// authoritative wire contract consumed by the cometline frontend. Each Kind
// marshals through a small typed struct so "type" stays first and field order is
// deterministic (a map would sort keys alphabetically and reorder the wire).
func (e Event) MarshalJSON() ([]byte, error) {
	t := string(e.Kind)
	switch e.Kind {
	case KindTextDelta:
		return json.Marshal(struct {
			Type  string `json:"type"`
			Delta string `json:"delta"`
		}{t, e.Delta})
	case KindReasoningDelta:
		return json.Marshal(struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}{t, e.Text})
	case KindToolCall:
		input := json.RawMessage(e.Input)
		if len(input) == 0 {
			input = json.RawMessage("{}")
		}
		return json.Marshal(struct {
			Type  string          `json:"type"`
			ID    string          `json:"id"`
			Tool  string          `json:"tool"`
			Input json.RawMessage `json:"input"`
		}{t, e.ID, e.Tool, input})
	case KindToolResult:
		return json.Marshal(struct {
			Type   string `json:"type"`
			ID     string `json:"id"`
			Tool   string `json:"tool"`
			Output string `json:"output"`
			Error  string `json:"error,omitempty"`
		}{t, e.ID, e.Tool, e.Output, e.ToolErr})
	case KindStepFinish:
		return json.Marshal(struct {
			Type  string `json:"type"`
			Usage Usage  `json:"usage"`
		}{t, e.Usage})
	case KindError:
		return json.Marshal(struct {
			Type    string `json:"type"`
			Message string `json:"message"`
			Code    string `json:"code,omitempty"`
		}{t, e.Message, e.Code})
	default:
		return json.Marshal(struct {
			Type string `json:"type"`
		}{t})
	}
}

// TextDelta builds a text_delta event.
func TextDelta(delta string) Event { return Event{Kind: KindTextDelta, Delta: delta} }

// ReasoningStart builds a reasoning_start event.
func ReasoningStart() Event { return Event{Kind: KindReasoningStart} }

// ReasoningDelta builds a reasoning_delta event.
func ReasoningDelta(text string) Event { return Event{Kind: KindReasoningDelta, Text: text} }

// ToolCall builds a tool_call event. input is JSON object bytes.
func ToolCall(id, tool string, input []byte) Event {
	return Event{Kind: KindToolCall, ID: id, Tool: tool, Input: input}
}

// ToolResult builds a tool_result event. toolErr is empty on success.
func ToolResult(id, tool, output, toolErr string) Event {
	return Event{Kind: KindToolResult, ID: id, Tool: tool, Output: output, ToolErr: toolErr}
}

// StepFinish builds a step_finish event from SDK token usage.
func StepFinish(u cometsdk.TokenUsage) Event {
	return Event{Kind: KindStepFinish, Usage: Usage{
		InputTokens:  u.InputTokens,
		OutputTokens: u.OutputTokens,
		CacheRead:    u.CacheRead,
		CacheWrite:   u.CacheWrite,
	}}
}

// Errorf builds an error event.
func Errorf(message, code string) Event {
	return Event{Kind: KindError, Message: message, Code: code}
}

// Done builds the terminal done event.
func Done() Event { return Event{Kind: KindDone} }
