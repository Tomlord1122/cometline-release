package openai

import (
	"encoding/json"
	"testing"

	cometsdk "github.com/cometline/comet-sdk"
	"github.com/stretchr/testify/require"
)

func TestConvertRequest_SystemPromptPrepended(t *testing.T) {
	req := &cometsdk.Request{
		Model:  "gpt-4o",
		System: "You are helpful.",
		Messages: []cometsdk.Message{
			{Role: cometsdk.RoleUser, Content: []cometsdk.Block{cometsdk.TextBlock{Text: "Hello"}}},
		},
	}

	data, err := toOpenAIRequest(req)
	require.NoError(t, err)

	var out openAIRequest
	require.NoError(t, json.Unmarshal(data, &out))

	require.Len(t, out.Messages, 2)
	require.Equal(t, "system", out.Messages[0].Role)
	require.Equal(t, "You are helpful.", out.Messages[0].Content)
	require.Equal(t, "user", out.Messages[1].Role)
}

func TestConvertRequest_ToolResultRole(t *testing.T) {
	req := &cometsdk.Request{
		Model: "gpt-4o",
		Messages: []cometsdk.Message{
			{
				Role: cometsdk.RoleToolResult,
				Content: []cometsdk.Block{
					cometsdk.ToolResultBlock{ToolCallID: "call_01", Content: "file contents"},
					cometsdk.ToolResultBlock{ToolCallID: "call_02", Content: "dir listing"},
				},
			},
		},
	}

	data, err := toOpenAIRequest(req)
	require.NoError(t, err)

	var out openAIRequest
	require.NoError(t, json.Unmarshal(data, &out))

	// Each ToolResultBlock becomes its own "tool" message.
	require.Len(t, out.Messages, 2)
	require.Equal(t, "tool", out.Messages[0].Role)
	require.Equal(t, "call_01", out.Messages[0].ToolCallID)
	require.Equal(t, "tool", out.Messages[1].Role)
	require.Equal(t, "call_02", out.Messages[1].ToolCallID)
}

func TestConvertRequest_AssistantToolCallsOnlyUsesNullContent(t *testing.T) {
	req := &cometsdk.Request{
		Model: "gpt-4o",
		Messages: []cometsdk.Message{
			{
				Role: cometsdk.RoleAssistant,
				Content: []cometsdk.Block{
					cometsdk.ToolCallBlock{
						ID:    "call_01",
						Name:  "list_dir",
						Input: json.RawMessage(`{"path":"."}`),
					},
				},
			},
		},
	}

	data, err := toOpenAIRequest(req)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(data, &out))

	msgs, ok := out["messages"].([]any)
	require.True(t, ok)
	require.Len(t, msgs, 1)

	msg, ok := msgs[0].(map[string]any)
	require.True(t, ok)
	require.Contains(t, msg, "content")
	require.Nil(t, msg["content"])

	toolCalls, ok := msg["tool_calls"].([]any)
	require.True(t, ok)
	require.Len(t, toolCalls, 1)
}

func TestConvertRequest_StreamOptionsIncludeUsage(t *testing.T) {
	req := &cometsdk.Request{
		Model:    "gpt-4o",
		Messages: []cometsdk.Message{{Role: cometsdk.RoleUser, Content: []cometsdk.Block{cometsdk.TextBlock{Text: "Hi"}}}},
	}
	data, err := toOpenAIRequest(req)
	require.NoError(t, err)

	var out openAIRequest
	require.NoError(t, json.Unmarshal(data, &out))

	require.NotNil(t, out.StreamOptions)
	require.True(t, out.StreamOptions.IncludeUsage)
}

func TestConvertEvent_MultipleToolCallsAssembled(t *testing.T) {
	state := newStreamState()

	// First tool call start.
	events, err := toSDKEvents(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_01","type":"function","function":{"name":"read_file","arguments":""}}]},"finish_reason":null}]}`, state)
	require.NoError(t, err)
	require.Len(t, events, 1)
	_, ok := events[0].(cometsdk.ToolCallStartEvent)
	require.True(t, ok)

	// Second tool call start.
	events, err = toSDKEvents(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"id":"call_02","type":"function","function":{"name":"list_dir","arguments":""}}]},"finish_reason":null}]}`, state)
	require.NoError(t, err)
	require.Len(t, events, 1)
	_, ok = events[0].(cometsdk.ToolCallStartEvent)
	require.True(t, ok)

	// Arguments for tool call 0.
	_, err = toSDKEvents(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":\"main.go\"}"}}]},"finish_reason":null}]}`, state)
	require.NoError(t, err)

	// Arguments for tool call 1.
	_, err = toSDKEvents(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":1,"function":{"arguments":"{\"path\":\"src\"}"}}]},"finish_reason":null}]}`, state)
	require.NoError(t, err)

	// finish_reason: tool_calls → ToolCallDoneEvents are emitted immediately;
	// StepFinishEvent is buffered until [DONE].
	events, err = toSDKEvents(`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`, state)
	require.NoError(t, err)

	// Flush StepFinishEvent via [DONE].
	doneLineEvents, err := toSDKEvents("[DONE]", state)
	require.NoError(t, err)
	events = append(events, doneLineEvents...)

	var doneEvents []cometsdk.ToolCallDoneEvent
	var stepFinish *cometsdk.StepFinishEvent
	for _, e := range events {
		switch ev := e.(type) {
		case cometsdk.ToolCallDoneEvent:
			doneEvents = append(doneEvents, ev)
		case cometsdk.StepFinishEvent:
			stepFinish = &ev
		}
	}

	require.Len(t, doneEvents, 2)
	require.NotNil(t, stepFinish)
	require.Equal(t, "tool_use", stepFinish.FinishReason)

	// Verify both tools are present (order not guaranteed from map iteration).
	names := map[string]bool{}
	for _, d := range doneEvents {
		names[d.Name] = true
	}
	require.True(t, names["read_file"])
	require.True(t, names["list_dir"])
}

func TestConvertEvent_FragmentedArguments(t *testing.T) {
	state := newStreamState()

	_, err := toSDKEvents(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_01","type":"function","function":{"name":"read_file","arguments":""}}]},"finish_reason":null}]}`, state)
	require.NoError(t, err)

	_, err = toSDKEvents(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":"}}]},"finish_reason":null}]}`, state)
	require.NoError(t, err)

	_, err = toSDKEvents(`{"choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"main.go\"}"}}]},"finish_reason":null}]}`, state)
	require.NoError(t, err)

	events, err := toSDKEvents(`{"choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}`, state)
	require.NoError(t, err)

	var done *cometsdk.ToolCallDoneEvent
	for _, e := range events {
		if d, ok := e.(cometsdk.ToolCallDoneEvent); ok {
			done = &d
		}
	}

	require.NotNil(t, done)
	require.JSONEq(t, `{"path":"main.go"}`, string(done.Input))
}

func TestConvertEvent_ReasoningContentAlias(t *testing.T) {
	state := newStreamState()

	events, err := toSDKEvents(`{"choices":[{"index":0,"delta":{"reasoning_content":"think"},"finish_reason":null}]}`, state)
	require.NoError(t, err)
	require.Len(t, events, 2)

	_, ok := events[0].(cometsdk.ReasoningStartEvent)
	require.True(t, ok)

	reasoning, ok := events[1].(cometsdk.ReasoningContentEvent)
	require.True(t, ok)
	require.Equal(t, "think", reasoning.Text)

	events, err = toSDKEvents("[DONE]", state)
	require.NoError(t, err)
	require.Len(t, events, 1)
	_, ok = events[0].(cometsdk.DoneEvent)
	require.True(t, ok)
}

// ─── Options passthrough tests ────────────────────────────────────────────────

func TestConvertRequest_OptionsPassthrough(t *testing.T) {
	// top_p, presence_penalty, temperature are common company API params.
	req := &cometsdk.Request{
		Model:     "gpt-4o",
		MaxTokens: 1000,
		Messages: []cometsdk.Message{
			{Role: cometsdk.RoleUser, Content: []cometsdk.Block{cometsdk.TextBlock{Text: "Hi"}}},
		},
		Options: map[string]any{
			"openai": map[string]any{
				"temperature":       0.8,
				"top_p":             1.0,
				"presence_penalty":  1.0,
				"frequency_penalty": 0.5,
			},
		},
	}

	data, err := toOpenAIRequest(req)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(data, &out))

	require.Equal(t, 0.8, out["temperature"])
	require.Equal(t, 1.0, out["top_p"])
	require.Equal(t, 1.0, out["presence_penalty"])
	require.Equal(t, 0.5, out["frequency_penalty"])

	// SDK-managed fields must not be overridden.
	require.Equal(t, "gpt-4o", out["model"])
	require.Equal(t, true, out["stream"])
}

func TestConvertRequest_OptionsDoNotOverrideSDKFields(t *testing.T) {
	// Caller tries to override model and stream — SDK fields must win.
	req := &cometsdk.Request{
		Model: "gpt-4o",
		Messages: []cometsdk.Message{
			{Role: cometsdk.RoleUser, Content: []cometsdk.Block{cometsdk.TextBlock{Text: "Hi"}}},
		},
		Options: map[string]any{
			"openai": map[string]any{
				"model":  "gpt-3.5-turbo", // should be ignored
				"stream": false,           // should be ignored
			},
		},
	}

	data, err := toOpenAIRequest(req)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(data, &out))

	require.Equal(t, "gpt-4o", out["model"])
	require.Equal(t, true, out["stream"])
}

func TestConvertRequest_OptionsNil(t *testing.T) {
	// No Options set — should behave identically to before.
	req := &cometsdk.Request{
		Model: "gpt-4o",
		Messages: []cometsdk.Message{
			{Role: cometsdk.RoleUser, Content: []cometsdk.Block{cometsdk.TextBlock{Text: "Hi"}}},
		},
	}

	data, err := toOpenAIRequest(req)
	require.NoError(t, err)

	var out map[string]any
	require.NoError(t, json.Unmarshal(data, &out))

	require.Equal(t, "gpt-4o", out["model"])
	require.Equal(t, true, out["stream"])
}

func TestConvertRequest_OptionsWrongType(t *testing.T) {
	// Options["openai"] is not map[string]any — should return an error.
	req := &cometsdk.Request{
		Model: "gpt-4o",
		Messages: []cometsdk.Message{
			{Role: cometsdk.RoleUser, Content: []cometsdk.Block{cometsdk.TextBlock{Text: "Hi"}}},
		},
		Options: map[string]any{
			"openai": "this-is-wrong",
		},
	}

	_, err := toOpenAIRequest(req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "map[string]any")
}
