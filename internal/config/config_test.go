package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadCreatesDefaultConfigWithBaseURL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Provider != ProviderAnthropic {
		t.Fatalf("Provider = %q, want %q", cfg.Provider, ProviderAnthropic)
	}
	if cfg.BaseURL != "" {
		t.Fatalf("BaseURL = %q, want empty", cfg.BaseURL)
	}

	path := filepath.Join(home, ".cometmind", "config.toml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected config file at %s: %v", path, err)
	}
}

func TestLoadReadsBaseURLEnvironmentOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("COMETMIND_BASE_URL", "http://localhost:11434/v1")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "http://localhost:11434/v1")
	}
}

func TestLoadReadsSystemPromptPathEnvironmentOverride(t *testing.T) {
	home := t.TempDir()
	promptPath := filepath.Join(home, "SOUL.md")
	t.Setenv("HOME", home)
	t.Setenv("COMETMIND_SYSTEM_PROMPT_PATH", promptPath)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.SystemPromptPath != promptPath {
		t.Fatalf("SystemPromptPath = %q, want %q", cfg.SystemPromptPath, promptPath)
	}
}

func TestLoadReadsProvidersArray(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	configDir := filepath.Join(home, ".cometmind")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `provider = "local-llm"
model = "qwen2.5"
base_url = "http://localhost:11434/v1"
max_tokens = 4096
max_steps = 25

[[providers]]
id = "local-llm"
name = "Local LLM"
method = "openai-compatible"
base_url = "http://localhost:11434/v1"
api_key = "ignored"
model = "qwen2.5"

[[providers]]
id = "anthropic"
name = "Anthropic"
method = "anthropic"
base_url = "https://api.anthropic.com"
api_key = "sk-ant-123"
model = "claude-sonnet-4-5"
`
	if err := os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(cfg.Providers) != 2 {
		t.Fatalf("len(Providers) = %d, want 2", len(cfg.Providers))
	}
	if cfg.FindProvider("local-llm") == nil {
		t.Fatal("expected to find provider 'local-llm'")
	}
	anthropic := cfg.FindProvider("anthropic")
	if anthropic == nil {
		t.Fatal("expected to find provider 'anthropic'")
	}
	if anthropic.APIKey != "sk-ant-123" {
		t.Fatalf("anthropic APIKey = %q, want %q", anthropic.APIKey, "sk-ant-123")
	}
}
