package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestBuildStdioTransportUsesAugmentedPath(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	bin := filepath.Join(home, ".local", "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	fakeDocker := filepath.Join(bin, "docker")
	if err := os.WriteFile(fakeDocker, []byte("#!/bin/sh\necho hi\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("PATH", "/usr/bin:/bin")

	transport, err := buildTransport(ServerConfig{
		ID:        "searxng",
		Transport: TransportStdio,
		Command:   "docker",
		Args:      []string{"version"},
	})
	if err != nil {
		t.Fatalf("buildTransport: %v", err)
	}

	cmdTransport, ok := transport.(*mcp.CommandTransport)
	if !ok {
		t.Fatalf("transport type = %T, want *mcp.CommandTransport", transport)
	}
	if cmdTransport.Command.Path != fakeDocker {
		t.Fatalf("command path = %q, want %q", cmdTransport.Command.Path, fakeDocker)
	}

	var pathValue string
	for _, kv := range cmdTransport.Command.Env {
		if strings.HasPrefix(kv, "PATH=") {
			pathValue = strings.TrimPrefix(kv, "PATH=")
			break
		}
	}
	if pathValue == "" {
		t.Fatal("PATH not set on MCP stdio command")
	}
	if !strings.Contains(pathValue, bin) {
		t.Fatalf("PATH %q missing %q", pathValue, bin)
	}
	if !strings.Contains(pathValue, "/usr/local/bin") {
		t.Fatalf("PATH %q missing /usr/local/bin", pathValue)
	}
}