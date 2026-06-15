package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/cometline/cometmind/internal/paths"
	"github.com/spf13/viper"
)

const (
	ProviderAnthropic    = "anthropic"
	ProviderOpenAI       = "openai"
	ProviderOpenAICompat = "openai-compatible"
	ProviderOpencodeGo   = "opencode-go"
)

// ProviderEntry is one configured LLM provider managed by Cometline.
type ProviderEntry struct {
	ID      string `mapstructure:"id"`
	Name    string `mapstructure:"name"`
	Method  string `mapstructure:"method"`
	BaseURL string `mapstructure:"base_url"`
	APIKey  string `mapstructure:"api_key"`
	Model   string `mapstructure:"model"`
}

// Config holds user-visible runtime settings loaded from ~/.cometmind/config.toml and environment.
type Config struct {
	Provider         string          `mapstructure:"provider"`
	Model            string          `mapstructure:"model"`
	BaseURL          string          `mapstructure:"base_url"`
	MaxTokens        int             `mapstructure:"max_tokens"`
	MaxSteps         int             `mapstructure:"max_steps"`
	SystemPromptPath string          `mapstructure:"system_prompt_path"`
	Providers        []ProviderEntry `mapstructure:"providers"`
}

// Defaults returns baseline values when the config file is missing keys.
func Defaults() *Config {
	return &Config{
		Provider:  ProviderAnthropic,
		Model:     "claude-sonnet-4-5",
		MaxTokens: 8192,
		MaxSteps:  50,
	}
}

// Load reads config from ~/.cometmind/config.toml (creating the parent dir), merges env, and unmarshals.
func Load() (*Config, error) {
	dataDir, err := paths.DataDir()
	if err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(dataDir, "config.toml")

	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigFile(cfgPath)

	// Environment
	v.SetEnvPrefix("COMETMIND")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	def := Defaults()
	v.SetDefault("provider", def.Provider)
	v.SetDefault("model", def.Model)
	v.SetDefault("base_url", def.BaseURL)
	v.SetDefault("max_tokens", def.MaxTokens)
	v.SetDefault("max_steps", def.MaxSteps)
	v.SetDefault("system_prompt_path", def.SystemPromptPath)

	if _, err := os.Stat(cfgPath); errors.Is(err, os.ErrNotExist) {
		if err := writeDefaultFile(cfgPath, def); err != nil {
			return nil, err
		}
	}

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	if c.Provider == "" {
		c.Provider = def.Provider
	}
	if c.Model == "" {
		c.Model = def.Model
	}
	if c.BaseURL == "" {
		c.BaseURL = def.BaseURL
	}
	if c.MaxTokens == 0 {
		c.MaxTokens = def.MaxTokens
	}
	if c.MaxSteps == 0 {
		c.MaxSteps = def.MaxSteps
	}
	if c.SystemPromptPath == "" {
		c.SystemPromptPath = def.SystemPromptPath
	}

	return &c, nil
}

// FindProvider returns the provider entry matching id, or nil if none exists.
func (c *Config) FindProvider(id string) *ProviderEntry {
	for i := range c.Providers {
		if c.Providers[i].ID == id {
			return &c.Providers[i]
		}
	}
	return nil
}

func writeDefaultFile(path string, def *Config) error {
	content := fmt.Sprintf(`# CometMind — https://github.com/cometline/cometmind
provider = %q
model = %q
base_url = %q
max_tokens = %d
max_steps = %d
system_prompt_path = %q
`, def.Provider, def.Model, def.BaseURL, def.MaxTokens, def.MaxSteps, def.SystemPromptPath)
	return os.WriteFile(path, []byte(content), 0o600)
}
