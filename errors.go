package cometsdk

import (
	"fmt"
	"time"
)

// AuthError is returned when the API key is invalid or missing (HTTP 401).
type AuthError struct {
	ProviderID string
	StatusCode int
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("cometsdk: %s: authentication failed (HTTP %d)", e.ProviderID, e.StatusCode)
}

// RateLimitError is returned on HTTP 429.
// RetryAfter indicates how long to wait before retrying (0 if not provided by the provider).
type RateLimitError struct {
	ProviderID string
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("cometsdk: %s: rate limited, retry after %s", e.ProviderID, e.RetryAfter)
	}
	return fmt.Sprintf("cometsdk: %s: rate limited", e.ProviderID)
}

// ServerError is returned on HTTP 5xx responses.
type ServerError struct {
	ProviderID string
	StatusCode int
	Message    string
}

func (e *ServerError) Error() string {
	return fmt.Sprintf("cometsdk: %s: server error (HTTP %d): %s", e.ProviderID, e.StatusCode, e.Message)
}

// ContextLengthError is returned when the input exceeds the model's context window.
type ContextLengthError struct {
	ProviderID string
	ModelID    string
}

func (e *ContextLengthError) Error() string {
	return fmt.Sprintf("cometsdk: %s: context length exceeded for model %s", e.ProviderID, e.ModelID)
}

// StreamError wraps an error that occurred mid-stream (after HTTP 200 was received).
type StreamError struct {
	ProviderID string
	Cause      error
}

func (e *StreamError) Error() string {
	return fmt.Sprintf("cometsdk: %s: stream error: %v", e.ProviderID, e.Cause)
}

func (e *StreamError) Unwrap() error {
	return e.Cause
}
