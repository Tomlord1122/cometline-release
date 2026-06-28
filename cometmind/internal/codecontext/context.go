package codecontext

import (
	"context"
	"fmt"
	"strings"

	cometsdk "github.com/cometline/comet-sdk"
)

// Query describes the user turn and workspace root used to retrieve relevant code.
type Query struct {
	WorkspacePath string
	Messages      []cometsdk.Message
}

// Block is a syntax-aware code fragment selected for the current turn.
type Block struct {
	Path      string
	Symbol    string
	Language  string
	StartLine int
	EndLine   int
	Content   string
	Score     float64
}

// Result contains the code context selected for prompt injection.
type Result struct {
	Blocks []Block
}

// Retriever selects relevant code context for a turn.
type Retriever interface {
	Retrieve(ctx context.Context, query Query) (Result, error)
}

// FormatPrompt renders retrieved code context as compact model-facing context.
func FormatPrompt(result Result) string {
	if len(result.Blocks) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n\nRelevant code context:\n")
	for _, block := range result.Blocks {
		if strings.TrimSpace(block.Content) == "" {
			continue
		}
		location := block.Path
		if block.StartLine > 0 && block.EndLine >= block.StartLine {
			location = fmt.Sprintf("%s:%d-%d", block.Path, block.StartLine, block.EndLine)
		}
		if block.Symbol != "" {
			b.WriteString(fmt.Sprintf("\n%s (%s):\n", location, block.Symbol))
		} else {
			b.WriteString(fmt.Sprintf("\n%s:\n", location))
		}
		b.WriteString("```")
		if block.Language != "" {
			b.WriteString(block.Language)
		}
		b.WriteByte('\n')
		b.WriteString(strings.TrimSpace(block.Content))
		b.WriteString("\n```\n")
	}
	return b.String()
}
