// Package anthropic implements the cometsdk.Provider interface for Anthropic's Messages API.
package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	cometsdk "github.com/cometline/comet-sdk"
	"github.com/cometline/comet-sdk/internal/retry"
)

const (
	defaultBaseURL     = "https://api.anthropic.com"
	anthropicVersion   = "2023-06-01"
	anthropicBetaTools = "tools-2024-04-04"
	providerID         = "anthropic"
)

// provider implements cometsdk.Provider for Anthropic.
type provider struct {
	apiKey string
	cfg    cometsdk.ProviderConfig
	log    *slog.Logger
}

// NewAnthropicProvider creates a Provider for Anthropic's Messages API.
// apiKey is required. Use cometsdk.WithBaseURL, cometsdk.WithHTTPClient,
// cometsdk.WithTimeout, cometsdk.WithMaxRetries, and cometsdk.WithLogger
// to override defaults.
func NewAnthropicProvider(apiKey string, opts ...cometsdk.Option) cometsdk.Provider {
	cfg := cometsdk.DefaultProviderConfig()
	cfg.BaseURL = defaultBaseURL
	for _, o := range opts {
		o(&cfg)
	}
	cfg.BaseURL = cometsdk.NormaliseBaseURL(cfg.BaseURL)
	log := cfg.Logger
	if log == nil {
		log = slog.New(noopHandler{})
	}
	return &provider{
		apiKey: apiKey,
		cfg:    cfg,
		log:    log.With("provider", providerID),
	}
}

func (p *provider) ID() string { return providerID }

// Stream sends req to the Anthropic Messages API and returns a channel of events.
// The channel is closed after DoneEvent or ErrorEvent.
func (p *provider) Stream(ctx context.Context, req *cometsdk.Request) (<-chan cometsdk.Event, error) {
	ch := make(chan cometsdk.Event, 32)

	p.log.DebugContext(ctx, "stream.start", "model", req.Model)

	attempt := 0
	var httpResp *http.Response

	err := retry.Do(ctx, p.cfg.MaxRetries, func() error {
		attempt++
		if attempt > 1 {
			p.log.DebugContext(ctx, "stream.retry", "attempt", attempt, "model", req.Model)
		}
		r, err := p.doRequest(ctx, req)
		if err != nil {
			p.log.DebugContext(ctx, "stream.request_error", "attempt", attempt, "error", err)
			return err
		}
		httpResp = r
		return nil
	}, isRetryable)

	if err != nil {
		p.log.DebugContext(ctx, "stream.failed", "error", err)
		return nil, err
	}

	go parseLoop(ctx, providerID, httpResp.Body, ch, p.log)
	return ch, nil
}

// doRequest builds and executes the HTTP request. It returns the raw response
// on HTTP 200, or a typed error for any non-200 status.
func (p *provider) doRequest(ctx context.Context, req *cometsdk.Request) (*http.Response, error) {
	body, err := toAnthropicRequest(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		anthropicEndpoint(p.cfg.BaseURL), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: build request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "text/event-stream")
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	httpReq.Header.Set("anthropic-beta", anthropicBetaTools)

	// Use Bearer auth when pointing at a unified gateway; native X-API-Key otherwise.
	if p.cfg.AuthMode == cometsdk.AuthModeBearer {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	} else {
		httpReq.Header.Set("X-API-Key", p.apiKey)
	}

	client := p.cfg.HTTPClient
	if p.cfg.Timeout > 0 {
		client = &http.Client{
			Transport: p.cfg.HTTPClient.Transport,
			Timeout:   p.cfg.Timeout,
		}
	}

	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: http: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, classifyHTTPError(providerID, resp, body)
	}

	return resp, nil
}

// classifyHTTPError maps an HTTP error response to a typed SDK error.
func classifyHTTPError(providerID string, resp *http.Response, body []byte) error {
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return &cometsdk.AuthError{ProviderID: providerID, StatusCode: resp.StatusCode}

	case http.StatusTooManyRequests:
		var d time.Duration
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil {
				d = time.Duration(secs) * time.Second
			}
		}
		return &cometsdk.RateLimitError{ProviderID: providerID, RetryAfter: d}

	default:
		msg := string(body)
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			msg = apiErr.Error.Message
		}
		return &cometsdk.ServerError{
			ProviderID: providerID,
			StatusCode: resp.StatusCode,
			Message:    msg,
		}
	}
}

// isRetryable returns true for errors that should be retried.
func isRetryable(err error) bool {
	switch e := err.(type) {
	case *cometsdk.RateLimitError:
		return true
	case *cometsdk.ServerError:
		return e.StatusCode == 500 || e.StatusCode == 502 ||
			e.StatusCode == 503 || e.StatusCode == 529
	}
	return false
}

func anthropicEndpoint(baseURL string) string {
	baseURL = cometsdk.NormaliseBaseURL(baseURL)
	if strings.HasSuffix(baseURL, "/v1") {
		return baseURL + "/messages"
	}
	return baseURL + "/v1/messages"
}

// noopHandler is a slog.Handler that discards all log records.
// Used when the caller passes WithLogger(nil).
type noopHandler struct{}

func (noopHandler) Enabled(_ context.Context, _ slog.Level) bool  { return false }
func (noopHandler) Handle(_ context.Context, _ slog.Record) error { return nil }
func (noopHandler) WithAttrs(_ []slog.Attr) slog.Handler          { return noopHandler{} }
func (noopHandler) WithGroup(_ string) slog.Handler               { return noopHandler{} }
