package codecontext

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cometsdk "github.com/cometline/comet-sdk"
)

func TestWorkspaceRetrieverFindsRelevantFunctionBlock(t *testing.T) {
	assertRelevantBlock(t, relevantBlockCase{fileName: "mini.go", query: "where is createMiniWindow defined?", symbol: "createMiniWindow", wantSnippet: "func createMiniWindow()", avoidSnippet: "unrelatedHelper", source: goSource})
}

func TestWorkspaceRetrieverFindsRelevantPythonFunctionBlock(t *testing.T) {
	assertRelevantBlock(t, relevantBlockCase{fileName: "mini.py", query: "where is create_mini_window defined?", symbol: "create_mini_window", wantSnippet: "def create_mini_window():", avoidSnippet: "unrelated_helper", source: pythonSource})
}

func TestWorkspaceRetrieverFindsRelevantJavaScriptFunctionBlock(t *testing.T) {
	assertRelevantBlock(t, relevantBlockCase{fileName: "mini.js", query: "where is createMiniWindow defined?", symbol: "createMiniWindow", wantSnippet: "function createMiniWindow()", avoidSnippet: "unrelatedHelper", source: javascriptSource})
}

func TestWorkspaceRetrieverFindsRelevantTypeScriptFunctionBlock(t *testing.T) {
	assertRelevantBlock(t, relevantBlockCase{fileName: "mini.ts", query: "where is createMiniWindow defined?", symbol: "createMiniWindow", wantSnippet: "function createMiniWindow()", avoidSnippet: "unrelatedHelper", source: typescriptSource})
}

func TestWorkspaceRetrieverFindsRelevantCFunctionBlock(t *testing.T) {
	assertRelevantBlock(t, relevantBlockCase{fileName: "mini.c", query: "where is create_mini_window defined?", symbol: "create_mini_window", wantSnippet: "char *create_mini_window()", avoidSnippet: "unrelated_helper", source: cSource})
}

func TestWorkspaceRetrieverFindsRelevantCppFunctionBlock(t *testing.T) {
	assertRelevantBlock(t, relevantBlockCase{fileName: "mini.cpp", query: "where is createMiniWindow defined?", symbol: "createMiniWindow", wantSnippet: "std::string createMiniWindow()", avoidSnippet: "unrelatedHelper", source: cppSource})
}

func TestWorkspaceRetrieverFindsRelevantYAMLMappingBlock(t *testing.T) {
	assertRelevantBlock(t, relevantBlockCase{fileName: "workflow.yaml", query: "where is miniWindow configured?", symbol: "miniWindow", wantSnippet: "miniWindow:", avoidSnippet: "unrelated:", source: yamlSource})
}

func TestWorkspaceRetrieverFindsRelevantSvelteFunctionBlock(t *testing.T) {
	assertRelevantBlock(t, relevantBlockCase{fileName: "MiniWindow.svelte", query: "where is createMiniWindow defined?", symbol: "createMiniWindow", wantSnippet: "function createMiniWindow()", avoidSnippet: "unrelatedHelper", source: svelteSource})
}

type relevantBlockCase struct {
	fileName     string
	query        string
	source       string
	symbol       string
	wantSnippet  string
	avoidSnippet string
}

func assertRelevantBlock(t *testing.T, tc relevantBlockCase) {
	t.Helper()
	root := t.TempDir()
	path := filepath.Join(root, tc.fileName)
	if err := os.WriteFile(path, []byte(tc.source), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	retriever := NewWorkspaceRetriever()

	result, err := retriever.Retrieve(context.Background(), Query{
		WorkspacePath: root,
		Messages: []cometsdk.Message{{
			Role:    cometsdk.RoleUser,
			Content: []cometsdk.Block{cometsdk.TextBlock{Text: tc.query}},
		}},
	})

	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}
	if len(result.Blocks) == 0 {
		t.Fatal("expected at least one code block")
	}
	block := result.Blocks[0]
	if block.Path != tc.fileName {
		t.Fatalf("Path = %q, want %s", block.Path, tc.fileName)
	}
	if block.Symbol != tc.symbol {
		t.Fatalf("Symbol = %q, want %s", block.Symbol, tc.symbol)
	}
	if !strings.Contains(block.Content, tc.wantSnippet) {
		t.Fatalf("block content missing target snippet %q:\n%s", tc.wantSnippet, block.Content)
	}
	if tc.avoidSnippet != "" && strings.Contains(block.Content, tc.avoidSnippet) {
		t.Fatalf("block content should not include unrelated snippet %q:\n%s", tc.avoidSnippet, block.Content)
	}
}

const goSource = `package mini

func unrelatedHelper() string {
	return "ignore me"
}

func createMiniWindow() string {
	return "mini"
}
`

const pythonSource = `def unrelated_helper():
    return "ignore me"

def create_mini_window():
    return "mini"
`

const javascriptSource = `function unrelatedHelper() {
  return "ignore me";
}

function createMiniWindow() {
  return "mini";
}
`

const typescriptSource = `function unrelatedHelper(): string {
  return "ignore me";
}

export function createMiniWindow(): string {
  return "mini";
}
`

const cSource = `char *unrelated_helper() {
  return "ignore me";
}

char *create_mini_window() {
  return "mini";
}
`

const cppSource = `#include <string>

std::string unrelatedHelper() {
  return "ignore me";
}

std::string createMiniWindow() {
  return "mini";
}
`

const yamlSource = `unrelated:
  enabled: false

miniWindow:
  enabled: true
  shortcut: Cmd+Shift+K
`

const svelteSource = `<script lang="ts">
  function unrelatedHelper() {
    return "ignore me";
  }

  function createMiniWindow() {
    return "mini";
  }
</script>

<button>{createMiniWindow()}</button>
`
