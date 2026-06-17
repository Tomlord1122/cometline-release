package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// RunCommand runs a shell command with cwd set to the workspace root.
type RunCommand struct{ Workspace Workspace }

func (RunCommand) Spec() ToolSpec {
	return ToolSpec{
		Name:        "run_command",
		Description: "Run a shell command. Working directory is the workspace root. Dangerous patterns are rejected.",
		Parameters:  json.RawMessage(`{"type":"object","properties":{"command":{"type":"string","description":"Command with arguments (shell interpretation)"}},"required":["command"]}`),
	}
}

func (r RunCommand) Execute(ctx context.Context, input json.RawMessage) (Result, error) {
	var in struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal(input, &in); err != nil {
		return Result{}, err
	}
	if err := denylistCheck(in.Command); err != nil {
		return Result{OK: false, Output: err.Error()}, nil
	}
	if _, err := r.Workspace.Resolve("."); err != nil {
		return Result{}, err
	}
	root := filepath.Clean(r.Workspace.Root)

	// Acquire a per-workspace mutex so concurrent sessions do not run
	// conflicting shell commands (e.g. git commit, go test) simultaneously
	// against the same workspace root.
	release := acquireWorkspaceLock(root)
	defer release()

	cmdCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "sh", "-c", in.Command) //nolint:gosec
	cmd.Dir = root

	out, err := cmd.CombinedOutput()
	text := string(out)

	var exit *int
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			c := ee.ExitCode()
			exit = &c
			return Result{OK: false, Output: text, ExitCode: exit}, nil
		}
		return Result{OK: false, Output: text + err.Error()}, nil
	}
	c := 0
	exit = &c
	return Result{OK: true, Output: text, ExitCode: exit}, nil
}

func denylistCheck(cmd string) error {
	c := strings.TrimSpace(strings.ToLower(cmd))
	patterns := []string{
		"curl ", "wget ", "ssh ", "scp ", "mkfs", "dd if=", "> /dev/", "sudo rm ", "rm -rf /",
	}
	for _, p := range patterns {
		if strings.Contains(c, p) {
			return fmt.Errorf("command rejected by safety deny-list (matched %q)", p)
		}
	}
	return nil
}
