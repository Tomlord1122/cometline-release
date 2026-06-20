package agent

import (
	"testing"

	"github.com/cometline/cometmind/internal/config"
)

func TestResolveContextWindow(t *testing.T) {
	if got := ResolveContextWindow(nil); got != defaultContextWindowLimit {
		t.Fatalf("nil config = %d, want %d", got, defaultContextWindowLimit)
	}
	if got := ResolveContextWindow(&config.Config{}); got != defaultContextWindowLimit {
		t.Fatalf("empty config = %d, want %d", got, defaultContextWindowLimit)
	}
	if got := ResolveContextWindow(&config.Config{ContextWindowLimit: contextWindowLimit256K}); got != contextWindowLimit256K {
		t.Fatalf("256k config = %d, want %d", got, contextWindowLimit256K)
	}
	if got := ResolveContextWindow(&config.Config{ContextWindowLimit: 999_999}); got != defaultContextWindowLimit {
		t.Fatalf("invalid config = %d, want %d", got, defaultContextWindowLimit)
	}
}
