package agent

import (
	"github.com/cometline/cometmind/internal/config"
)

const (
	defaultContextWindowLimit = 128_000
	contextWindowLimit256K    = 256_000
)

// ResolveContextWindow returns the user-configured compaction budget limit.
// Only 128k and 256k are supported; anything else falls back to 128k.
func ResolveContextWindow(cfg *config.Config) int {
	if cfg == nil {
		return defaultContextWindowLimit
	}
	if cfg.ContextWindowLimit == contextWindowLimit256K {
		return contextWindowLimit256K
	}
	return defaultContextWindowLimit
}
