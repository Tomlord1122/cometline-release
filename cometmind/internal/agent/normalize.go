package agent

import (
	"strings"

	cometsdk "github.com/cometline/comet-sdk"
	"github.com/cometline/cometmind/internal/config"
)

// HistoryDegradation describes a single class of change normalization applied to
// the session history before replaying it to a provider. It is surfaced to
// callers (and ultimately the UI) so users understand why a switched provider
// may behave differently than the one that produced the history.
type HistoryDegradation struct {
	// Kind is a stable machine-readable identifier (e.g. "reasoning_summarized").
	Kind string
	// Count is how many history messages were affected.
	Count int
	// Message is a short human-readable explanation.
	Message string
}

// reasoningReplayMode classifies how a comet-sdk provider family handles
// assistant reasoning (chain-of-thought) content carried in history.
type reasoningReplayMode int

const (
	// reasoningNative: the provider re-ingests reasoning content as-is.
	reasoningNative reasoningReplayMode = iota
	// reasoningSummarize: reasoning must be degraded to plain summary text
	// because the provider rejects or drops raw reasoning on replay.
	reasoningSummarize
)

// reasoningModeForFamily maps a comet-sdk provider family (as returned by
// provider.SDKFamily) to its reasoning replay behaviour.
func reasoningModeForFamily(family string) reasoningReplayMode {
	switch family {
	case config.ProviderCodex:
		// The Codex Responses adapter does not replay assistant reasoning
		// content; raw chain-of-thought from a prior provider would be dropped
		// silently, so we degrade it to a visible summary instead.
		return reasoningSummarize
	default:
		// Anthropic and OpenAI replay reasoning content natively.
		return reasoningNative
	}
}

// reasoningSummaryPrefix is prepended to summarized reasoning so it is clearly
// distinguishable from the assistant's final text in the replayed transcript.
const reasoningSummaryPrefix = "[prior reasoning] "

// NormalizeHistoryForProvider adapts session history so it can be safely
// replayed to the target provider family. It returns a possibly-rewritten copy
// of messages plus a list of degradations describing any lossy adaptations.
//
// family is the comet-sdk provider family (config.ProviderAnthropic,
// config.ProviderOpenAI, or config.ProviderCodex). The input slice is never
// mutated; messages that need no change are shared by reference.
func NormalizeHistoryForProvider(family string, messages []cometsdk.Message) ([]cometsdk.Message, []HistoryDegradation) {
	mode := reasoningModeForFamily(family)
	sanitized, droppedEmpty := dropEmptyAssistantMessages(messages)
	if mode == reasoningNative {
		return sanitized, historyDegradations(0, droppedEmpty)
	}

	out := make([]cometsdk.Message, len(sanitized))
	summarized := 0
	for i, m := range sanitized {
		if m.Role != cometsdk.RoleAssistant || len(m.ReasoningContent) == 0 {
			out[i] = m
			continue
		}

		summary := summarizeReasoning(m.ReasoningContent)
		if summary == "" {
			// Nothing salvageable; just drop the reasoning channel.
			out[i] = cometsdk.Message{Role: m.Role, Content: m.Content}
			summarized++
			continue
		}

		// Prepend the summary as a visible text block so the model still sees a
		// trace of prior reasoning without replaying raw chain-of-thought.
		content := make([]cometsdk.Block, 0, len(m.Content)+1)
		content = append(content, cometsdk.TextBlock{Text: reasoningSummaryPrefix + summary})
		content = append(content, m.Content...)
		out[i] = cometsdk.Message{Role: m.Role, Content: content}
		summarized++
	}

	return out, historyDegradations(summarized, droppedEmpty)
}

func historyDegradations(summarized, droppedEmpty int) []HistoryDegradation {
	var degradations []HistoryDegradation
	if summarized > 0 {
		degradations = append(degradations, HistoryDegradation{
			Kind:    "reasoning_summarized",
			Count:   summarized,
			Message: "Prior reasoning was condensed into summary text because the selected provider cannot replay chain-of-thought.",
		})
	}
	if droppedEmpty > 0 {
		degradations = append(degradations, HistoryDegradation{
			Kind:    "empty_assistant_dropped",
			Count:   droppedEmpty,
			Message: "Empty assistant turns were skipped because some providers reject replaying assistant messages without text or tool calls.",
		})
	}
	return degradations
}

func dropEmptyAssistantMessages(messages []cometsdk.Message) ([]cometsdk.Message, int) {
	dropped := 0
	for _, m := range messages {
		if isEmptyAssistantMessage(m) {
			// A prior provider may have ended the stream without yielding text,
			// reasoning, or tool calls. Replaying that row into OpenAI-compatible
			// APIs triggers 400s because assistant history must contain content.
			dropped++
		}
	}
	if dropped == 0 {
		return messages, 0
	}
	out := make([]cometsdk.Message, 0, len(messages)-dropped)
	for _, m := range messages {
		if isEmptyAssistantMessage(m) {
			continue
		}
		out = append(out, m)
	}
	return out, dropped
}

func isEmptyAssistantMessage(m cometsdk.Message) bool {
	return m.Role == cometsdk.RoleAssistant && len(m.Content) == 0 && len(m.ReasoningContent) == 0
}

// summarizeReasoning collapses reasoning blocks into a single trimmed line.
func summarizeReasoning(blocks []cometsdk.Block) string {
	var parts []string
	for _, b := range blocks {
		switch rb := b.(type) {
		case cometsdk.ReasoningBlock:
			if t := strings.TrimSpace(rb.Text); t != "" {
				parts = append(parts, t)
			}
		case cometsdk.TextBlock:
			if t := strings.TrimSpace(rb.Text); t != "" {
				parts = append(parts, t)
			}
		}
	}
	joined := strings.Join(parts, " ")
	// Collapse internal whitespace/newlines so the summary stays a single line.
	joined = strings.Join(strings.Fields(joined), " ")
	return joined
}
