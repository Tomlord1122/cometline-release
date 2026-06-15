// Package runtime is the shared composition root for CometMind commands.
//
// It owns config loading, SQLite opening, and the wiring that turns a
// persisted session into a runnable agent. Commands become thin: they call
// runtime.New, ask it for whatever service they need, and defer Close.
package runtime

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"

	cometsdk "github.com/cometline/comet-sdk"
	"github.com/cometline/cometmind/internal/agent"
	"github.com/cometline/cometmind/internal/config"
	"github.com/cometline/cometmind/internal/paths"
	"github.com/cometline/cometmind/internal/provider"
	"github.com/cometline/cometmind/internal/session"
	"github.com/cometline/cometmind/internal/store"
	"github.com/cometline/cometmind/internal/tools"
)

// Runtime is the composition root shared by the CLI and server.
type Runtime struct {
	Config       *config.Config
	DB           *sql.DB
	Sessions     *session.Service
	SystemPrompt string
}

// New builds a Runtime from the environment and filesystem.
func New(ctx context.Context) (*Runtime, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	systemPrompt, err := loadSystemPrompt(cfg.SystemPromptPath)
	if err != nil {
		return nil, err
	}

	dbpath, err := paths.DBPath()
	if err != nil {
		return nil, fmt.Errorf("db path: %w", err)
	}
	sqlDB, err := store.OpenSQLite(ctx, dbpath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	r := &Runtime{
		Config:       cfg,
		DB:           sqlDB,
		Sessions:     session.New(sqlDB),
		SystemPrompt: systemPrompt,
	}
	return r, nil
}

func loadSystemPrompt(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read system prompt %q: %w", path, err)
	}
	return strings.TrimSpace(string(raw)), nil
}

// Close releases runtime resources.
func (r *Runtime) Close() error {
	return r.DB.Close()
}

// WorkspaceForCommand resolves the current directory (or the explicit workspace
// flag when passed) to a persisted workspace.
func (r *Runtime) WorkspaceForCommand(ctx context.Context, explicitWorkspace string) (session.Workspace, error) {
	root, err := paths.ResolveWorkspace(explicitWorkspace)
	if err != nil {
		return session.Workspace{}, fmt.Errorf("workspace root: %w", err)
	}
	return r.Sessions.EnsureWorkspace(ctx, root)
}

// ProviderForSession builds a provider configured for the given session's
// model/provider identifiers. The runtime's base config is copied so per-session
// overrides do not leak back into the global config.
func (r *Runtime) ProviderForSession(sess session.Session) (cometsdk.Provider, error) {
	cfg := *r.Config
	cfg.Model = sess.ModelID
	return provider.NewFor(&cfg, sess.ProviderID)
}

// RunnerFor returns an agent runner wired for a specific session and workspace.
func (r *Runtime) RunnerFor(sess session.Session, workspacePath string) (*agent.Runner, error) {
	p, err := r.ProviderForSession(sess)
	if err != nil {
		return nil, err
	}

	return &agent.Runner{
		Provider:     p,
		Sessions:     r.Sessions,
		Registry:     tools.NewRegistry(workspacePath),
		MaxSteps:     r.Config.MaxSteps,
		MaxTokens:    r.Config.MaxTokens,
		SystemPrompt: r.SystemPrompt,
	}, nil
}
