# Comet SDK — High-Level Design Document

> A provider-agnostic Go LLM client library. The Go equivalent of Vercel AI SDK.

| Field        | Value                                      |
|--------------|--------------------------------------------|
| Version      | 0.1.0 (Draft)                              |
| Module       | `github.com/cometline/comet-sdk`           |
| Authors      | Cometline Team                             |
| Status       | Proposed                                   |
| Last Updated | 2026-02-23                                 |

---

## Table of Contents

1. [Background](#1-background)
2. [Design Principles](#2-design-principles)
3. [Public Interface](#3-public-interface)
4. [Provider Implementations](#4-provider-implementations)
5. [Internal Architecture](#5-internal-architecture)
6. [Error Handling & Retry](#6-error-handling--retry)
7. [Testing Strategy](#7-testing-strategy)
8. [Package Layout](#8-package-layout)
9. [Tech Stack](#9-tech-stack)

---

## 1. Background

### 1.1 The Problem

The TypeScript ecosystem has **Vercel AI SDK** — a single library that abstracts over 20+ LLM providers behind a unified interface. It handles:
- Streaming responses (SSE)
- Tool calling / function calling
- Provider-specific message normalisation
- Retry on transient errors
- Token usage tracking

**Go has no equivalent.** The current options are:

| Library | Problem |
|---------|---------|
| `github.com/sashabaranov/go-openai` | OpenAI only; no streaming tool calls |
| `github.com/anthropics/anthropic-sdk-go` | Anthropic only |
| Community wrappers | Inconsistent API, often unmaintained |

Every Go project that needs to talk to more than one LLM provider ends up writing its own HTTP client and SSE parser from scratch.

### 1.2 What Comet SDK Is

Comet SDK is a **pure Go HTTP client library** for LLM APIs. It provides:

- A single `Provider` interface that works the same way for Anthropic, OpenAI, and Ollama
- A unified `Event` stream — one channel of typed events regardless of provider
- Automatic message normalisation to handle provider-specific quirks
- SSE parsing, retry logic, and token tracking built in

It is used internally by CometCode (the Cometline coding agent server), but is designed as a standalone Go module that any Go project can import.

### 1.3 What Comet SDK Is Not

- **Not a server framework.** It makes outbound HTTP requests. Gin, Echo, Chi have nothing to do with it.
- **Not an agent framework.** It has no concept of sessions, tools, or agent loops. That is CometCode's responsibility.
- **Not a prompt library.** It sends whatever messages the caller provides.
- **Not a model registry.** It does not know which models exist; the caller specifies `providerID` + `modelID`.

### 1.4 Why Build It Instead of Using Existing Libraries

The goal of the Cometline project is to understand how LLM tool-calling and streaming work at the HTTP level. Writing the provider clients directly — rather than wrapping an existing SDK — is the engineering learning goal. The secondary benefit is that the result is a reusable library with no opaque abstractions.

---

## 2. Design Principles

### 2.1 Unified Interface, Provider-Specific Internals

The public interface is identical for every provider. All provider-specific differences (different JSON field names, different SSE event types, different tool call ID constraints) are handled inside each provider's `convert.go`. The caller never needs to know which provider it is using.

### 2.2 Streaming First

Every LLM call returns a `<-chan Event`. There is no non-streaming path. This matches how modern LLM APIs work and avoids duplicating logic for streaming vs non-streaming modes.

### 2.3 No External Dependencies

Comet SDK's core depends only on the Go standard library:
- `net/http` — HTTP client
- `bufio` — SSE line scanning
- `encoding/json` — request/response serialisation
- `context` — cancellation

The only allowed external dependency is a testing helper (`testify`) which is a `require` only in `go.mod`'s test section.

### 2.4 Errors Are Typed

Every error returned by Comet SDK is a concrete type, not a string-wrapped `fmt.Errorf`. Callers can `errors.As` to inspect the cause and act accordingly (e.g. retry on `RateLimitError`, surface to user on `AuthError`).

### 2.5 Context Cancellation Is First-Class

Every `Stream()` call accepts a `context.Context`. If the context is cancelled (user hits abort, timeout), the HTTP request is cancelled and the event channel is closed cleanly. No goroutine leaks.

---

## 3. Public Interface

### 3.1 Core Types

```go
package cometsdk

// Provider is the single interface implemented by every LLM backend.
// Callers only ever interact with this interface — never with concrete types.
type Provider interface {
    // ID returns the provider identifier, e.g. "anthropic", "openai", "ollama".
    ID() string

    // Stream sends a request to the LLM and returns a channel of Events.
    // The channel is closed when the LLM finishes or the context is cancelled.
    // Errors during streaming are sent as ErrorEvent before the channel closes.
    Stream(ctx context.Context, req *Request) (<-chan Event, error)
}
```

### 3.2 Request

```go
// Request is the provider-agnostic input to Stream().
type Request struct {
    // Model is the model identifier as the provider expects it,
    // e.g. "claude-sonnet-4-5", "gpt-4o", "llama3.2".
    Model string

    // Messages is the conversation history.
    Messages []Message

    // Tools is the list of tools the LLM may call.
    // Empty slice means no tool calling.
    Tools []Tool

    // System is the system prompt. Empty string means no system prompt.
    System string

    // MaxTokens caps the output. 0 means use the provider default.
    MaxTokens int

    // Temperature controls randomness. nil means use the provider default.
    Temperature *float64

    // Options holds provider-specific overrides, e.g.
    // {"anthropic": {"cache_control": "ephemeral"}}
    // Most callers leave this nil.
    Options map[string]any
}
```

### 3.3 Message

```go
// Message is a single turn in the conversation.
type Message struct {
    Role    Role     // RoleUser | RoleAssistant | RoleToolResult
    Content []Block  // one or more content blocks
}

type Role string

const (
    RoleUser       Role = "user"
    RoleAssistant  Role = "assistant"
    RoleToolResult Role = "tool_result"
)

// Block is a sealed interface — only the types below implement it.
type Block interface{ isBlock() }

// TextBlock is a plain text content block.
type TextBlock struct {
    Text string
}

// ToolCallBlock represents a tool invocation emitted by the assistant.
type ToolCallBlock struct {
    ID    string          // provider-assigned call ID
    Name  string          // tool name
    Input json.RawMessage // JSON-encoded arguments
}

// ToolResultBlock is the result of a tool call, sent back by the caller.
type ToolResultBlock struct {
    ToolCallID string // matches ToolCallBlock.ID
    Content    string // tool output (text)
    IsError    bool   // true if the tool failed
}
```

### 3.4 Tool

```go
// Tool describes a function the LLM can call.
// Parameters must be a valid JSON Schema object.
type Tool struct {
    Name        string
    Description string
    Parameters  json.RawMessage // JSON Schema: {"type":"object","properties":{...}}
}
```

### 3.5 Events

```go
// Event is a sealed interface. All event types below implement it.
type Event interface{ isEvent() }

// TextDeltaEvent carries a streamed text token from the LLM.
type TextDeltaEvent struct {
    Text string
}

// ToolCallStartEvent signals the LLM has started a tool call.
// Input may be partial at this point; wait for ToolCallDoneEvent for complete input.
type ToolCallStartEvent struct {
    ID   string
    Name string
}

// ToolCallDeltaEvent carries a partial JSON fragment of the tool input.
type ToolCallDeltaEvent struct {
    ID    string
    Delta string
}

// ToolCallDoneEvent signals the tool call input is complete.
type ToolCallDoneEvent struct {
    ID    string
    Name  string
    Input json.RawMessage // complete, valid JSON
}

// StepFinishEvent is emitted at the end of each LLM call (one "step").
type StepFinishEvent struct {
    FinishReason string     // "stop" | "tool_use" | "max_tokens" | "error"
    Usage        TokenUsage
}

// ErrorEvent carries a non-fatal streaming error.
// After an ErrorEvent the channel is closed.
type ErrorEvent struct {
    Err error
}

// DoneEvent is the final event. After this the channel is closed.
type DoneEvent struct{}
```

### 3.6 TokenUsage

```go
type TokenUsage struct {
    InputTokens  int
    OutputTokens int
    CacheRead    int // prompt cache read tokens (Anthropic)
    CacheWrite   int // prompt cache write tokens (Anthropic)
}
```

### 3.7 Constructor Functions

```go
// NewAnthropicProvider creates a provider for Anthropic's Messages API.
// apiKey is required. baseURL is optional (defaults to https://api.anthropic.com).
func NewAnthropicProvider(apiKey string, opts ...Option) Provider

// NewOpenAIProvider creates a provider for OpenAI's Chat Completions API.
func NewOpenAIProvider(apiKey string, opts ...Option) Provider

// NewOllamaProvider creates a provider for a local Ollama instance.
// baseURL defaults to http://localhost:11434.
func NewOllamaProvider(baseURL string, opts ...Option) Provider

// Option is a functional option for provider configuration.
type Option func(*providerConfig)

func WithBaseURL(url string) Option
func WithHTTPClient(client *http.Client) Option
func WithTimeout(d time.Duration) Option
func WithMaxRetries(n int) Option
```

### 3.8 Usage Example

```go
provider := cometsdk.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY"))

req := &cometsdk.Request{
    Model:  "claude-sonnet-4-5",
    System: "You are a helpful coding assistant.",
    Messages: []cometsdk.Message{
        {
            Role:    cometsdk.RoleUser,
            Content: []cometsdk.Block{cometsdk.TextBlock{Text: "Fix the bug in main.go"}},
        },
    },
    Tools: []cometsdk.Tool{
        {
            Name:        "read_file",
            Description: "Read the contents of a file",
            Parameters:  json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
        },
    },
}

events, err := provider.Stream(ctx, req)
if err != nil {
    log.Fatal(err)
}

for event := range events {
    switch e := event.(type) {
    case cometsdk.TextDeltaEvent:
        fmt.Print(e.Text)
    case cometsdk.ToolCallDoneEvent:
        fmt.Printf("\n[tool call: %s(%s)]\n", e.Name, e.Input)
    case cometsdk.StepFinishEvent:
        fmt.Printf("\n[tokens: %d in, %d out]\n", e.Usage.InputTokens, e.Usage.OutputTokens)
    case cometsdk.ErrorEvent:
        log.Printf("stream error: %v", e.Err)
    }
}
```

---

## 4. Provider Implementations

### 4.1 Anthropic

**API:** `POST https://api.anthropic.com/v1/messages`
**Streaming format:** SSE with typed event names

#### Request Mapping

| Comet SDK field | Anthropic field |
|-----------------|-----------------|
| `Request.Model` | `model` |
| `Request.System` | `system` (top-level string) |
| `Request.Messages` | `messages` array |
| `Request.Tools` | `tools` array with `input_schema` |
| `Request.MaxTokens` | `max_tokens` |
| `TextBlock` | `{"type":"text","text":"..."}` |
| `ToolCallBlock` | `{"type":"tool_use","id":"...","name":"...","input":{...}}` |
| `ToolResultBlock` | `{"type":"tool_result","tool_use_id":"...","content":"..."}` |

#### SSE Event Mapping

| Anthropic SSE event | Comet SDK Event |
|--------------------|-----------------|
| `content_block_start` with `type: text` | — (buffer start) |
| `content_block_delta` with `type: text_delta` | `TextDeltaEvent` |
| `content_block_start` with `type: tool_use` | `ToolCallStartEvent` |
| `content_block_delta` with `type: input_json_delta` | `ToolCallDeltaEvent` |
| `content_block_stop` (after tool_use) | `ToolCallDoneEvent` |
| `message_delta` with `stop_reason` | `StepFinishEvent` |
| `message_stop` | `DoneEvent` |

#### Normalisation Quirks (from OpenCode analysis)

- Anthropic rejects messages where `content` is an empty array — filter these out before sending.
- Tool call IDs must match `[a-zA-Z0-9_-]` — sanitise any other characters.
- Required headers: `anthropic-version: 2023-06-01`, `anthropic-beta: tools-2024-04-04`.

### 4.2 OpenAI

**API:** `POST https://api.openai.com/v1/chat/completions`
**Streaming format:** SSE, each `data:` line is a JSON delta object, terminated by `data: [DONE]`

#### Request Mapping

| Comet SDK field | OpenAI field |
|-----------------|--------------|
| `Request.Model` | `model` |
| `Request.System` | `{"role":"system","content":"..."}` prepended to messages |
| `Request.Messages` | `messages` array |
| `Request.Tools` | `tools` array with `function.parameters` |
| `Request.MaxTokens` | `max_tokens` |
| `TextBlock` | `{"type":"text","text":"..."}` or string content |
| `ToolCallBlock` | `tool_calls[].function` |
| `ToolResultBlock` | `{"role":"tool","tool_call_id":"...","content":"..."}` |

#### SSE Event Mapping

| OpenAI delta field | Comet SDK Event |
|--------------------|-----------------|
| `choices[0].delta.content` (non-null) | `TextDeltaEvent` |
| `choices[0].delta.tool_calls[].function.name` (new index) | `ToolCallStartEvent` |
| `choices[0].delta.tool_calls[].function.arguments` | `ToolCallDeltaEvent` |
| `choices[0].finish_reason == "tool_calls"` | `ToolCallDoneEvent` (assembled from deltas) |
| `choices[0].finish_reason == "stop"` | `StepFinishEvent` + `DoneEvent` |
| `data: [DONE]` | channel close |

#### Normalisation Quirks

- Tool call arguments arrive as fragmented JSON strings across multiple deltas — must be assembled into a complete JSON object before emitting `ToolCallDoneEvent`.
- Multiple tool calls can be in-flight simultaneously (different `index` values) — track each independently.
- `usage` field only appears on the final delta when `stream_options: {"include_usage": true}` is set — always set this.

### 4.3 Ollama

**API:** `POST http://localhost:11434/api/chat` (OpenAI-compatible mode)

Ollama supports an OpenAI-compatible endpoint at `/v1/chat/completions`. The Ollama provider is a thin wrapper around the OpenAI provider with a different base URL and no API key requirement:

```go
func NewOllamaProvider(baseURL string, opts ...Option) Provider {
    if baseURL == "" {
        baseURL = "http://localhost:11434"
    }
    return NewOpenAIProvider("ollama", // dummy key
        WithBaseURL(baseURL+"/v1"),
        append(opts, withOllamaQuirks())...,
    )
}
```

Ollama-specific differences:
- No `Authorization` header needed (or accepted).
- Some models do not support tool calling — `ToolCallDoneEvent` may never arrive; the agent loop must handle this gracefully.
- Token usage fields may be zero for some models.

---

## 5. Internal Architecture

### 5.1 Directory Layout

```
comet-sdk/
├── sdk.go                  # All public types and interfaces (Provider, Request,
│                           # Message, Block, Event, TokenUsage, Option)
├── errors.go               # All public error types
├── provider/
│   ├── anthropic/
│   │   ├── client.go       # NewAnthropicProvider(), providerConfig, HTTP setup
│   │   ├── convert.go      # Request → Anthropic JSON; Anthropic events → sdk.Event
│   │   └── stream.go       # SSE reader loop, sends Events to channel
│   ├── openai/
│   │   ├── client.go       # NewOpenAIProvider(), HTTP setup
│   │   ├── convert.go      # Request → OpenAI JSON; OpenAI deltas → sdk.Event
│   │   │                   # (handles multi-delta tool call assembly)
│   │   └── stream.go       # SSE reader loop
│   └── ollama/
│       └── client.go       # Thin wrapper around openai/ with adjusted defaults
├── internal/
│   ├── sse/
│   │   └── scanner.go      # Reusable SSE line scanner (bufio.Scanner wrapper)
│   └── retry/
│       └── retry.go        # Exponential backoff with jitter
└── docs/
    └── HLD.md              # This document
```

### 5.2 Stream Lifecycle

```
caller                    provider                      LLM API
──────                    ────────                      ───────
Stream(ctx, req)
    │
    ├─► convert request to provider JSON
    ├─► http.NewRequestWithContext(ctx, POST, url, body)
    ├─► client.Do(req)  ──────────────────────────────► HTTP POST
    │                                                    │
    │   make(chan Event, 32)  ◄── response body (SSE) ───┘
    │   go parseLoop(body, ch)
    │
    └─► return ch, nil
         │
         │   [goroutine]
         ├─► scanner.Scan() → line
         ├─► parse line → raw event
         ├─► convert.toEvent(raw) → typed sdk.Event
         ├─► ch <- event
         │   (repeat until done or ctx cancelled)
         │
         ├─► ch <- DoneEvent{}
         └─► close(ch)
```

### 5.3 Tool Call Assembly (OpenAI)

OpenAI streams tool call arguments as fragments across multiple SSE deltas. The stream parser maintains an in-progress map:

```go
type inProgressToolCall struct {
    id        string
    name      string
    argBuffer strings.Builder
}

// map key = tool call index from OpenAI delta
inProgress := map[int]*inProgressToolCall{}

// On each delta:
for _, tc := range delta.ToolCalls {
    if tc.Function.Name != "" {
        // new tool call starting
        inProgress[tc.Index] = &inProgressToolCall{id: tc.ID, name: tc.Function.Name}
        ch <- ToolCallStartEvent{ID: tc.ID, Name: tc.Function.Name}
    }
    if tc.Function.Arguments != "" {
        inProgress[tc.Index].argBuffer.WriteString(tc.Function.Arguments)
        ch <- ToolCallDeltaEvent{ID: inProgress[tc.Index].id, Delta: tc.Function.Arguments}
    }
}

// On finish_reason == "tool_calls":
for _, tc := range inProgress {
    input := json.RawMessage(tc.argBuffer.String())
    ch <- ToolCallDoneEvent{ID: tc.id, Name: tc.name, Input: input}
}
```

### 5.4 Message Normalisation

Each provider's `convert.go` is responsible for normalising outgoing messages. Key rules:

**Anthropic:**
```go
// Remove messages with empty content arrays
msgs = filterEmptyContent(msgs)

// Sanitise tool call IDs: only [a-zA-Z0-9_-] allowed
msgs = sanitiseToolCallIDs(msgs)
```

**OpenAI:**
```go
// System prompt becomes first message with role "system"
// Tool results become messages with role "tool"
// Multiple tool calls in one assistant turn are all in tool_calls[]
```

---

## 6. Error Handling & Retry

### 6.1 Error Types

```go
// AuthError is returned when the API key is invalid or missing.
type AuthError struct {
    ProviderID string
    StatusCode int
}

// RateLimitError is returned on HTTP 429.
// RetryAfter indicates when to retry (if the provider sends a Retry-After header).
type RateLimitError struct {
    ProviderID string
    RetryAfter time.Duration
}

// ServerError is returned on HTTP 5xx.
type ServerError struct {
    ProviderID string
    StatusCode int
    Message    string
}

// ContextLengthError is returned when the input exceeds the model's context window.
type ContextLengthError struct {
    ProviderID string
    ModelID    string
}

// StreamError wraps an error that occurred mid-stream (after HTTP 200).
type StreamError struct {
    ProviderID string
    Cause      error
}
```

### 6.2 Retry Policy

Retryable errors: `RateLimitError`, `ServerError` (500, 502, 503, 529).

Non-retryable: `AuthError`, `ContextLengthError`, any HTTP 4xx other than 429.

```
attempt 1: immediate
attempt 2: 1s  + jitter
attempt 3: 2s  + jitter
attempt 4: 4s  + jitter
(max 4 attempts by default, configurable via WithMaxRetries)
```

If `RateLimitError.RetryAfter > 0`, use that duration instead of the backoff schedule.

Context cancellation always stops retrying immediately.

### 6.3 Mid-stream Errors

If an error occurs after the HTTP response has started (i.e. after the channel is returned to the caller), the error is sent as an `ErrorEvent` and the channel is closed. The caller should treat `ErrorEvent` as terminal.

---

## 7. Testing Strategy

### 7.1 Unit Tests — convert.go

Test the request conversion and event normalisation logic in isolation, with no HTTP calls:

```
provider/anthropic/convert_test.go
  TestConvertRequest_basic
  TestConvertRequest_toolCall
  TestConvertRequest_emptyContentFiltered      ← Anthropic quirk
  TestConvertRequest_toolCallIDSanitised       ← Anthropic quirk
  TestConvertEvent_textDelta
  TestConvertEvent_toolCallDone
  TestConvertEvent_stepFinish

provider/openai/convert_test.go
  TestConvertRequest_systemPromptPrepended
  TestConvertRequest_toolResultRole
  TestConvertEvent_multipleToolCallsAssembled  ← OpenAI streaming quirk
  TestConvertEvent_fragmentedArguments
```

### 7.2 Integration Tests — stream.go

Use a test HTTP server (`httptest.NewServer`) that replays a canned SSE fixture:

```
provider/anthropic/stream_test.go
  TestStream_textOnly          ← fixture: text response, no tools
  TestStream_toolCall          ← fixture: one tool call then stop
  TestStream_contextCancelled  ← cancel ctx mid-stream, verify no goroutine leak
  TestStream_rateLimitRetry    ← 429 on first attempt, 200 on second

provider/openai/stream_test.go
  TestStream_textOnly
  TestStream_multipleToolCalls ← two tool calls in one step
  TestStream_contextCancelled
```

### 7.3 No Live API Calls in CI

All tests use `httptest.NewServer` with recorded fixtures. No `ANTHROPIC_API_KEY` or `OPENAI_API_KEY` needed in CI. Live API tests can be run locally with a build tag: `go test -tags=live ./...`.

---

## 8. Package Layout

```
comet-sdk/
│
├── go.mod                          # module github.com/cometline/comet-sdk
│                                   # go 1.23
│                                   # require: (nothing — stdlib only)
│                                   # require (test): github.com/stretchr/testify
│
├── sdk.go                          # package cometsdk
│                                   # Provider, Request, Message, Role,
│                                   # Block (TextBlock, ToolCallBlock, ToolResultBlock),
│                                   # Event (all event types),
│                                   # Tool, TokenUsage, Option
│
├── errors.go                       # AuthError, RateLimitError, ServerError,
│                                   # ContextLengthError, StreamError
│
├── provider/
│   ├── anthropic/
│   │   ├── client.go               # NewAnthropicProvider()
│   │   ├── convert.go              # toAnthropicRequest(), toSDKEvent()
│   │   ├── convert_test.go
│   │   ├── stream.go               # parseLoop()
│   │   ├── stream_test.go
│   │   └── fixtures/               # Canned SSE responses for tests
│   │       ├── text_response.sse
│   │       └── tool_call.sse
│   │
│   ├── openai/
│   │   ├── client.go               # NewOpenAIProvider()
│   │   ├── convert.go              # toOpenAIRequest(), toSDKEvent(), assembleToolCalls()
│   │   ├── convert_test.go
│   │   ├── stream.go
│   │   ├── stream_test.go
│   │   └── fixtures/
│   │       ├── text_response.sse
│   │       └── tool_calls_multi.sse
│   │
│   └── ollama/
│       ├── client.go               # NewOllamaProvider() — wraps openai/
│       └── client_test.go
│
├── internal/
│   ├── sse/
│   │   ├── scanner.go              # SSE line scanner, shared by all providers
│   │   └── scanner_test.go
│   └── retry/
│       ├── retry.go                # Do(ctx, fn, policy) — exponential backoff
│       └── retry_test.go
│
└── docs/
    └── HLD.md                      # This document
```

---

## 9. Tech Stack

| Component       | Choice                    | Rationale                                                                 |
|-----------------|---------------------------|---------------------------------------------------------------------------|
| Language        | **Go 1.23+**              | Module consumers need no special runtime; single binary output.           |
| HTTP client     | **`net/http`** (stdlib)   | No external dependency needed; LLM APIs are standard HTTPS POST.         |
| SSE parsing     | **`bufio.Scanner`** (stdlib) | Line-by-line scanning of `data: ...` events. Simple and allocation-efficient. |
| JSON            | **`encoding/json`** (stdlib) | Request/response serialisation. `json.RawMessage` for tool call inputs.  |
| Context         | **`context`** (stdlib)    | Cancellation propagated from caller to HTTP request to goroutine.        |
| Testing         | **`testing`** (stdlib) + **`testify/require`** | Standard Go test runner + clean assertion helpers. |
| Test server     | **`net/http/httptest`** (stdlib) | Replay SSE fixtures without live API calls.                        |
| No server framework | — | Comet SDK is an HTTP **client** library. Gin, Echo, and similar server frameworks are irrelevant here. |

---

*End of document.*
