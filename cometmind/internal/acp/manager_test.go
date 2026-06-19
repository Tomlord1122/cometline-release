package acp_test

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/cometline/cometmind/internal/acp"
)

func TestSessionManagerRun(t *testing.T) {
	t.Parallel()

	mgr := acp.NewSessionManager(acp.DefaultConfig())
	mgr.ProcessStarter = func(ctx context.Context, cfg acp.Config) (io.WriteCloser, io.ReadCloser, io.Closer, error) {
		return acp.StartFakeAgentPipes(ctx)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := mgr.Run(ctx, acp.RunOptions{
		ChildSessionID: "child-1",
		WorkspaceRoot:  t.TempDir(),
		Task:           "fix tests",
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if result.Status != "completed" {
		t.Fatalf("status = %q", result.Status)
	}
	if !strings.Contains(result.Summary, "fix tests") {
		t.Fatalf("summary = %q", result.Summary)
	}
}
