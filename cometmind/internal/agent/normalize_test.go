package agent

import (
	"encoding/json"
	"strings"
	"testing"

	cometsdk "github.com/cometline/comet-sdk"
	"github.com/cometline/cometmind/internal/config"
)

func assistantWithReasoning() cometsdk.Message {
	return cometsdk.Message{
		Role: cometsdk.RoleAssistant,
		Content: []cometsdk.Block{
			cometsdk.TextBlock{Text: "final answer"},
			cometsdk.ToolCallBlock{ID: "c1", Name: "read", Input: json.RawMessage(`{"path":"main.go"}`)},
		},
		ReasoningContent: []cometsdk.Block{
			cometsdk.ReasoningBlock{Text: "let me think\nstep by step"},
		},
	}
}

func TestNormalizeHistoryForProvider_NativeFamiliesUnchanged(t *testing.T) {
	in := []cometsdk.Message{assistantWithReasoning()}
	for _, family := range []string{config.ProviderAnthropic, config.ProviderOpenAI} {
		out, degradations := NormalizeHistoryForProvider(family, in)
		if len(degradations) != 0 {
			t.Fatalf("%s: expected no degradations, got %v", family, degradations)
		}
		if len(out[0].ReasoningContent) != 1 {
			t.Fatalf("%s: reasoning content was modified", family)
		}
		if len(out[0].Content) != 2 {
			t.Fatalf("%s: content was modified, got %d blocks", family, len(out[0].Content))
		}
	}
}

func TestNormalizeHistoryForProvider_DropsEmptyAssistantForAllFamilies(t *testing.T) {
	in := []cometsdk.Message{
		{Role: cometsdk.RoleUser, Content: []cometsdk.Block{cometsdk.TextBlock{Text: "hi"}}},
		{Role: cometsdk.RoleAssistant},
		{Role: cometsdk.RoleAssistant, Content: []cometsdk.Block{cometsdk.TextBlock{Text: "hello"}}},
	}

	for _, family := range []string{config.ProviderAnthropic, config.ProviderOpenAI, config.ProviderCodex} {
		out, degradations := NormalizeHistoryForProvider(family, in)
		if len(out) != 2 {
			t.Fatalf("%s: expected empty assistant dropped, got %d messages", family, len(out))
		}
		if len(degradations) == 0 || degradations[len(degradations)-1].Kind != "empty_assistant_dropped" {
			t.Fatalf("%s: expected empty_assistant_dropped degradation, got %+v", family, degradations)
		}
	}
}

func TestNormalizeHistoryForProvider_CodexSummarizesReasoning(t *testing.T) {
	in := []cometsdk.Message{assistantWithReasoning()}
	out, degradations := NormalizeHistoryForProvider(config.ProviderCodex, in)

	if len(degradations) != 1 || degradations[0].Kind != "reasoning_summarized" {
		t.Fatalf("expected reasoning_summarized degradation, got %+v", degradations)
	}
	if degradations[0].Count != 1 {
		t.Fatalf("expected count 1, got %d", degradations[0].Count)
	}

	msg := out[0]
	if len(msg.ReasoningContent) != 0 {
		t.Fatalf("expected reasoning content cleared, got %d blocks", len(msg.ReasoningContent))
	}
	if len(msg.Content) != 3 {
		t.Fatalf("expected summary prepended (3 blocks), got %d", len(msg.Content))
	}
	first, ok := msg.Content[0].(cometsdk.TextBlock)
	if !ok {
		t.Fatalf("expected first block TextBlock, got %T", msg.Content[0])
	}
	if !strings.HasPrefix(first.Text, reasoningSummaryPrefix) {
		t.Fatalf("summary missing prefix: %q", first.Text)
	}
	if !strings.Contains(first.Text, "let me think step by step") {
		t.Fatalf("summary should collapse whitespace: %q", first.Text)
	}
	// The original final text and tool call must be preserved after the summary.
	if _, ok := msg.Content[2].(cometsdk.ToolCallBlock); !ok {
		t.Fatalf("expected tool call preserved, got %T", msg.Content[2])
	}
}

func TestNormalizeHistoryForProvider_CodexDoesNotMutateInput(t *testing.T) {
	in := []cometsdk.Message{assistantWithReasoning()}
	NormalizeHistoryForProvider(config.ProviderCodex, in)
	if len(in[0].ReasoningContent) != 1 {
		t.Fatalf("input slice was mutated: reasoning len = %d", len(in[0].ReasoningContent))
	}
	if len(in[0].Content) != 2 {
		t.Fatalf("input slice was mutated: content len = %d", len(in[0].Content))
	}
}

func TestNormalizeHistoryForProvider_CodexEmptyReasoningDropped(t *testing.T) {
	in := []cometsdk.Message{{
		Role:             cometsdk.RoleAssistant,
		Content:          []cometsdk.Block{cometsdk.TextBlock{Text: "hi"}},
		ReasoningContent: []cometsdk.Block{cometsdk.ReasoningBlock{Text: "   "}},
	}}
	out, degradations := NormalizeHistoryForProvider(config.ProviderCodex, in)
	if len(degradations) != 1 {
		t.Fatalf("expected a degradation, got %v", degradations)
	}
	if len(out[0].ReasoningContent) != 0 {
		t.Fatalf("expected reasoning dropped")
	}
	if len(out[0].Content) != 1 {
		t.Fatalf("expected no summary block for empty reasoning, got %d", len(out[0].Content))
	}
}

func TestNormalizeHistoryForProvider_NonAssistantUntouched(t *testing.T) {
	in := []cometsdk.Message{
		{Role: cometsdk.RoleUser, Content: []cometsdk.Block{cometsdk.TextBlock{Text: "q"}}},
	}
	out, degradations := NormalizeHistoryForProvider(config.ProviderCodex, in)
	if len(degradations) != 0 {
		t.Fatalf("expected no degradations for user message, got %v", degradations)
	}
	if len(out[0].Content) != 1 {
		t.Fatalf("user message altered")
	}
}
