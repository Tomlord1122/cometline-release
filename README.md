# Comet SDK

A provider-agnostic Go LLM client library. One interface, any backend.

```
module: github.com/cometline/comet-sdk
go:     1.25
```

---

## Role in the Cometline stack

Comet SDK is the **LLM I/O layer** of the Cometline stack. It is consumed by **CometMind** (the general AI agent runtime), giving it uniform model access regardless of provider.

```
┌─────────────────────────────────────────────┐
│  cometline   Electron desktop shell           │  ← UI
├─────────────────────────────────────────────┤
│  cometmind   General AI agent runtime         │  ← brain
│              (agent loop, tools, persistence; │
│               delegates coding to OpenCode    │
│               via ACP)                        │
├─────────────────────────────────────────────┤
│  comet-sdk   Provider-agnostic LLM I/O        │  ← this repo
└─────────────────────────────────────────────┘
```

Comet SDK deliberately stays a **pure LLM I/O library**: it does not contain an agent loop, does not execute tools, and does not persist anything. That separation lets CometMind own orchestration while the SDK focuses solely on talking to models cleanly across providers.

---

## What it does

Comet SDK gives you a single `Provider` interface that works identically for Anthropic, OpenAI, and any OpenAI-compatible endpoint (e.g. a company unified API). It handles:

- Streaming responses over SSE
- Tool calling with full delta assembly
- Provider-specific message normalisation
- Automatic retry with exponential backoff
- Token usage tracking
- Structured debug logging via `log/slog`

For most callers, the recommended public entry point is the `llm` package, especially `llm.GenerateMessage` and `llm.StreamMessage`. Use `Provider.Stream()` directly when you need lower-level control over raw events.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         caller                              │
│                                                             │
│   provider.Stream(ctx, &Request{                            │
│       Model, Messages, Tools, System,                       │
│       MaxTokens, Options["openai"|"anthropic"]              │
│   })                                                        │
└──────────────────────────┬──────────────────────────────────┘
                           │  <-chan Event
                           ▼
┌─────────────────────────────────────────────────────────────┐
│                   cometsdk  (sdk.go)                        │
│                                                             │
│  Provider · Request · Message · Block                       │
│  Event (TextDelta · ToolCallStart · ToolCallDelta ·         │
│         ToolCallDone · StepFinish · Error · Done)           │
│  Tool · TokenUsage · ProviderConfig · Option                │
│                                                             │
│  errors.go                                                  │
│  AuthError · RateLimitError · ServerError ·                 │
│  ContextLengthError · StreamError                           │
└────────────────┬──────────────────────┬─────────────────────┘
                 │                      │
    ┌────────────▼────────┐  ┌──────────▼───────────┐
    │  provider/anthropic  │  │   provider/openai     │
    │                      │  │                       │
    │  client.go           │  │  client.go            │
    │  convert.go          │  │  convert.go           │
    │  stream.go           │  │  stream.go            │
    │  fixtures/           │  │  fixtures/            │
    └────────┬─────────────┘  └──────────┬────────────┘
             │                           │
             └─────────────┬─────────────┘
                           │
           ┌───────────────▼───────────────┐
           │          internal/            │
           │                               │
           │  sse/scanner.go               │
           │  bufio.Scanner wrapper        │
           │  parses event: / data: lines  │
           │                               │
           │  retry/retry.go               │
           │  cenkalti/backoff/v4          │
           │  1s → 2s → 4s + jitter        │
           │  Retry-After · ctx cancel     │
           └───────────────────────────────┘
                        │
          ┌─────────────┼──────────────────┐
          ▼             ▼                  ▼
   Anthropic API   OpenAI API      Company Unified API
   /v1/messages    /v1/chat/       (OpenAI-compatible)
                   completions     WithBaseURL(...)
```

---

## Installation

```bash
go get github.com/cometline/comet-sdk
```

---

## Quick start

### Recommended: `llm.GenerateMessage`

```go
import (
    "context"
    "encoding/json"
    "fmt"
    "os"

    cometsdk "github.com/cometline/comet-sdk"
    "github.com/cometline/comet-sdk/llm"
    "github.com/cometline/comet-sdk/provider/anthropic"
)

p := anthropic.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY"))

req := &cometsdk.Request{
    Model:  "claude-sonnet-4-5",
    System: "You are a helpful coding assistant.",
    Messages: []cometsdk.Message{
        {
            Role:    cometsdk.RoleUser,
            Content: []cometsdk.Block{cometsdk.TextBlock{Text: "Read main.go and tell me what it does."}},
        },
    },
    Tools: []cometsdk.Tool{
        {
            Name:        "read_file",
            Description: "Read the contents of a file",
            Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
        },
    },
    MaxTokens: 1024,
}

result, err := llm.GenerateMessage(context.Background(), p, req)
if err != nil {
    panic(err)
}

fmt.Println("finish reason:", result.FinishReason)
for _, tc := range result.ToolCalls {
    fmt.Printf("tool: %s input=%s\n", tc.Name, tc.Input)
}
```

### Recommended streaming: `llm.StreamMessage`

```go
import (
    "context"
    "fmt"
    "os"

    cometsdk "github.com/cometline/comet-sdk"
    "github.com/cometline/comet-sdk/llm"
    "github.com/cometline/comet-sdk/provider/openai"
)

p := openai.NewOpenAIProvider(
    os.Getenv("CUSTOM_API_KEY"),
    cometsdk.WithBaseURL("https://your-company-api.example.com"),
)

req := &cometsdk.Request{
    Model:     "gpt-4o",
    MaxTokens: 1000,
    Messages: []cometsdk.Message{
        {
            Role:    cometsdk.RoleUser,
            Content: []cometsdk.Block{cometsdk.TextBlock{Text: "What is the capital of France?"}},
        },
    },
    // Provider-specific parameters via Options
    Options: map[string]any{
        "openai": map[string]any{
            "temperature":      0.8,
            "top_p":            1.0,
            "presence_penalty": 1.0,
        },
    },
}

stream := llm.StreamMessage(context.Background(), p, req)

for ev := range stream.Events() {
    switch e := ev.(type) {
    case cometsdk.TextDeltaEvent:
        fmt.Print(e.Text)
    case cometsdk.ToolCallStartEvent:
        fmt.Printf("\n[start tool %s]\n", e.Name)
    case cometsdk.ToolCallDoneEvent:
        fmt.Printf("\n[done tool %s input=%s]\n", e.Name, e.Input)
    }
}

result, err := stream.Result()
if err != nil {
    panic(err)
}

fmt.Printf("\nfinish=%s tool_calls=%d\n", result.FinishReason, len(result.ToolCalls))
```

### Lower-level primitive: `Provider.Stream()`

Use `Provider.Stream()` directly when you want raw provider-normalized events and will assemble the response yourself.

```go
ch, err := p.Stream(context.Background(), req)
if err != nil {
    panic(err)
}

for event := range ch {
    switch e := event.(type) {
    case cometsdk.TextDeltaEvent:
        fmt.Print(e.Text)
    case cometsdk.ToolCallDoneEvent:
        fmt.Printf("\n[tool: %s(%s)]\n", e.Name, e.Input)
    case cometsdk.StepFinishEvent:
        fmt.Printf("\n[tokens: %d in / %d out]\n", e.Usage.InputTokens, e.Usage.OutputTokens)
    case cometsdk.ErrorEvent:
        panic(e.Err)
    }
}
```

---

## Provider-specific options

Any parameter not in the `Request` struct can be passed through `Options` without changing SDK code:

```go
// Anthropic — thinking, top_k, top_p, etc.
Options: map[string]any{
    "anthropic": map[string]any{
        "top_k": 40,
        "thinking": map[string]any{
            "type":          "enabled",
            "budget_tokens": 5000,
        },
    },
}

// OpenAI — temperature, top_p, presence_penalty, seed, etc.
Options: map[string]any{
    "openai": map[string]any{
        "temperature":       0.8,
        "top_p":             1.0,
        "presence_penalty":  1.0,
        "frequency_penalty": 0.5,
    },
}
```

SDK-managed fields (`model`, `messages`, `stream`, `max_tokens`) cannot be overridden via Options.

---

## Configuration options

```go
p := anthropic.NewAnthropicProvider(apiKey,
    cometsdk.WithBaseURL("https://custom-endpoint.example.com"),
    cometsdk.WithTimeout(30 * time.Second),
    cometsdk.WithMaxRetries(3),
    cometsdk.WithHTTPClient(myHTTPClient),
    cometsdk.WithLogger(slog.Default()),
)
```

### Debug logging

```go
log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))
p := anthropic.NewAnthropicProvider(apiKey, cometsdk.WithLogger(log))
```

Pass `cometsdk.WithLogger(nil)` to silence all SDK output.

---

## Error handling

All errors are concrete types, inspectable with `errors.As`:

```go
_, err := llm.GenerateMessage(ctx, p, req)
if err != nil {
    var rle *cometsdk.RateLimitError
    if errors.As(err, &rle) {
        time.Sleep(rle.RetryAfter)
    }
}
```

| Error type | When |
|---|---|
| `AuthError` | Invalid or missing API key (401/403) |
| `RateLimitError` | Rate limited (429); carries `RetryAfter` |
| `ServerError` | Provider 5xx response |
| `ContextLengthError` | Input exceeds model context window |
| `StreamError` | Error occurring mid-stream (after HTTP 200) |

---

## Testing

```bash
make test               # go test ./..., no API calls, CI-safe
make test-verbose       # same with -v
make test-anthropic     # Anthropic package only
make test-openai        # OpenAI package only
make test-live          # real API calls (requires env vars)
make test-live-anthropic
make test-live-openai
```

**Live test environment variables:**

```bash
# Company unified API (supports both Anthropic /v1/messages and OpenAI /v1/chat/completions)
export CUSTOM_API_KEY="..."
export CUSTOM_BASE_URL="https://your-company-api.example.com"
# also valid:
export CUSTOM_BASE_URL="https://your-company-api.example.com/v1"

# Direct provider access (used as fallback when CUSTOM_API_KEY is not set)
export ANTHROPIC_API_KEY="sk-ant-..."   # fallback for Anthropic live tests
export OPENAI_API_KEY="sk-..."          # fallback for OpenAI live tests

# CUSTOM_BASE_URL applies to any provider and accepts either a root URL or
# a /v1-suffixed URL:
#   Anthropic → replaces https://api.anthropic.com
#   OpenAI    → replaces https://api.openai.com
```

---

## Dependencies

| Package | Purpose |
|---|---|
| `github.com/cenkalti/backoff/v4` | Exponential backoff retry (runtime) |
| `github.com/stretchr/testify` | Test assertions (test only) |
| `log/slog` | Structured logging (stdlib) |

---

## Documentation

- [`docs/HLD.md`](docs/HLD.md) — High-level design document
