package codecontext

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode"

	cometsdk "github.com/cometline/comet-sdk"
	sittersvelte "github.com/tree-sitter-grammars/tree-sitter-svelte/bindings/go"
	sitteryaml "github.com/tree-sitter-grammars/tree-sitter-yaml/bindings/go"
	sitter "github.com/tree-sitter/go-tree-sitter"
	sitterc "github.com/tree-sitter/tree-sitter-c/bindings/go"
	sittercpp "github.com/tree-sitter/tree-sitter-cpp/bindings/go"
	sittergo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	sitterjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	sitterpython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	sittertypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

const maxSourceBytes = 256 * 1024

type WorkspaceRetriever struct{}

func NewWorkspaceRetriever() *WorkspaceRetriever {
	return &WorkspaceRetriever{}
}

func (r *WorkspaceRetriever) Retrieve(ctx context.Context, query Query) (Result, error) {
	terms := queryTerms(query)
	if query.WorkspacePath == "" || len(terms) == 0 {
		return Result{}, nil
	}
	var blocks []Block
	err := filepath.WalkDir(query.WorkspacePath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		name := entry.Name()
		if entry.IsDir() {
			if shouldSkipDir(name) {
				return filepath.SkipDir
			}
			return nil
		}
		isSvelte := strings.EqualFold(filepath.Ext(name), ".svelte")
		lang, ok := languageForPath(name)
		if !ok && !isSvelte {
			return nil
		}
		if !ok {
			lang = parserLanguage{}
		}
		info, err := entry.Info()
		if err != nil || info.Size() > maxSourceBytes {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(query.WorkspacePath, path)
		if err != nil {
			rel = path
		}
		if isSvelte {
			blocks = append(blocks, extractSvelteBlocks(rel, source, terms)...)
			return nil
		}
		blocks = append(blocks, extractBlocks(rel, source, terms, lang)...)
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	sort.SliceStable(blocks, func(i, j int) bool {
		return blocks[i].Score > blocks[j].Score
	})
	if len(blocks) > 5 {
		blocks = blocks[:5]
	}
	return Result{Blocks: blocks}, nil
}

func shouldSkipDir(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "node_modules", "vendor", "dist", "build", ".next", "out", "coverage", "__pycache__":
		return true
	default:
		return false
	}
}

type parserLanguage struct {
	name      string
	language  *sitter.Language
	nodeKinds map[string]bool
}

func languageForPath(path string) (parserLanguage, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return parserLanguage{
			name:     "go",
			language: sitter.NewLanguage(sittergo.Language()),
			nodeKinds: map[string]bool{
				"function_declaration": true,
				"method_declaration":   true,
			},
		}, true
	case ".py":
		return parserLanguage{
			name:     "python",
			language: sitter.NewLanguage(sitterpython.Language()),
			nodeKinds: map[string]bool{
				"function_definition": true,
				"class_definition":    true,
			},
		}, true
	case ".js", ".jsx":
		return parserLanguage{
			name:     "javascript",
			language: sitter.NewLanguage(sitterjavascript.Language()),
			nodeKinds: map[string]bool{
				"function_declaration": true,
				"class_declaration":    true,
			},
		}, true
	case ".ts":
		return parserLanguage{
			name:     "typescript",
			language: sitter.NewLanguage(sittertypescript.LanguageTypescript()),
			nodeKinds: map[string]bool{
				"function_declaration": true,
				"class_declaration":    true,
			},
		}, true
	case ".tsx":
		return parserLanguage{
			name:     "tsx",
			language: sitter.NewLanguage(sittertypescript.LanguageTSX()),
			nodeKinds: map[string]bool{
				"function_declaration": true,
				"class_declaration":    true,
			},
		}, true
	case ".c", ".h":
		return parserLanguage{
			name:     "c",
			language: sitter.NewLanguage(sitterc.Language()),
			nodeKinds: map[string]bool{
				"function_definition": true,
			},
		}, true
	case ".cpp", ".cc", ".cxx", ".hpp", ".hh", ".hxx":
		return parserLanguage{
			name:     "cpp",
			language: sitter.NewLanguage(sittercpp.Language()),
			nodeKinds: map[string]bool{
				"function_definition": true,
			},
		}, true
	case ".yaml", ".yml":
		return parserLanguage{
			name:     "yaml",
			language: sitter.NewLanguage(sitteryaml.Language()),
			nodeKinds: map[string]bool{
				"block_mapping_pair": true,
			},
		}, true
	default:
		return parserLanguage{}, false
	}
}

func extractBlocks(path string, source []byte, terms []string, lang parserLanguage) []Block {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(lang.language); err != nil {
		return nil
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return nil
	}
	defer tree.Close()
	var blocks []Block
	var visit func(*sitter.Node)
	visit = func(node *sitter.Node) {
		if node == nil {
			return
		}
		if lang.nodeKinds[node.Kind()] {
			if block, ok := blockFromNode(path, source, node, terms, lang.name); ok {
				blocks = append(blocks, block)
			}
		}
		for i := uint(0); i < node.NamedChildCount(); i++ {
			visit(node.NamedChild(i))
		}
	}
	visit(tree.RootNode())
	return blocks
}

func extractSvelteBlocks(path string, source []byte, terms []string) []Block {
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(sitter.NewLanguage(sittersvelte.Language())); err != nil {
		return nil
	}
	tree := parser.Parse(source, nil)
	if tree == nil {
		return nil
	}
	tree.Close()

	for _, script := range svelteScriptContents(source) {
		lang := parserLanguage{
			name:     "svelte",
			language: sitter.NewLanguage(sittertypescript.LanguageTSX()),
			nodeKinds: map[string]bool{
				"function_declaration": true,
				"class_declaration":    true,
			},
		}
		if blocks := extractBlocks(path, script, terms, lang); len(blocks) > 0 {
			return blocks
		}
	}
	return nil
}

func svelteScriptContents(source []byte) [][]byte {
	text := string(source)
	lower := strings.ToLower(text)
	var scripts [][]byte
	searchFrom := 0
	for {
		start := strings.Index(lower[searchFrom:], "<script")
		if start < 0 {
			break
		}
		start += searchFrom
		openEnd := strings.Index(lower[start:], ">")
		if openEnd < 0 {
			break
		}
		contentStart := start + openEnd + 1
		closeStart := strings.Index(lower[contentStart:], "</script>")
		if closeStart < 0 {
			break
		}
		contentEnd := contentStart + closeStart
		scripts = append(scripts, []byte(text[contentStart:contentEnd]))
		searchFrom = contentEnd + len("</script>")
	}
	return scripts
}

func blockFromNode(path string, source []byte, node *sitter.Node, terms []string, language string) (Block, bool) {
	symbol := nodeSymbol(node, source)
	if symbol == "" {
		return Block{}, false
	}
	content := strings.TrimSpace(node.Utf8Text(source))
	if symbol == "" || content == "" {
		return Block{}, false
	}
	score := scoreBlock(path, symbol, content, terms)
	if score <= 0 {
		return Block{}, false
	}
	return Block{
		Path:      filepath.ToSlash(path),
		Symbol:    symbol,
		Language:  language,
		StartLine: int(node.StartPosition().Row) + 1,
		EndLine:   int(node.EndPosition().Row) + 1,
		Content:   content,
		Score:     score,
	}, true
}

func nodeSymbol(node *sitter.Node, source []byte) string {
	if name := node.ChildByFieldName("name"); name != nil {
		return cleanSymbol(name.Utf8Text(source))
	}
	if key := node.ChildByFieldName("key"); key != nil {
		return cleanSymbol(key.Utf8Text(source))
	}
	if declarator := node.ChildByFieldName("declarator"); declarator != nil {
		return firstIdentifier(declarator, source)
	}
	return ""
}

func cleanSymbol(symbol string) string {
	return strings.Trim(strings.TrimSpace(symbol), `"'`)
}

func firstIdentifier(node *sitter.Node, source []byte) string {
	if node == nil {
		return ""
	}
	if node.Kind() == "identifier" || node.Kind() == "field_identifier" {
		return strings.TrimSpace(node.Utf8Text(source))
	}
	for i := uint(0); i < node.NamedChildCount(); i++ {
		if symbol := firstIdentifier(node.NamedChild(i), source); symbol != "" {
			return symbol
		}
	}
	return ""
}

func scoreBlock(path, symbol, content string, terms []string) float64 {
	lowerPath := strings.ToLower(path)
	lowerSymbol := strings.ToLower(symbol)
	lowerContent := strings.ToLower(content)
	var score float64
	for _, term := range terms {
		if term == "" {
			continue
		}
		if lowerSymbol == term {
			score += 10
		} else if strings.Contains(lowerSymbol, term) || strings.Contains(term, lowerSymbol) {
			score += 6
		}
		if strings.Contains(lowerPath, term) {
			score += 2
		}
		if strings.Contains(lowerContent, term) {
			score += 1
		}
	}
	return score
}

func queryTerms(query Query) []string {
	seen := map[string]bool{}
	var out []string
	for _, msg := range query.Messages {
		for _, block := range msg.Content {
			appendTerms(blockText(block), seen, &out)
		}
	}
	return out
}

func blockText(block any) string {
	switch b := block.(type) {
	case cometsdk.TextBlock:
		return b.Text
	case *cometsdk.TextBlock:
		if b == nil {
			return ""
		}
		return b.Text
	case cometsdk.ReasoningBlock:
		return b.Text
	case *cometsdk.ReasoningBlock:
		if b == nil {
			return ""
		}
		return b.Text
	default:
		return ""
	}
}

func appendTerms(text string, seen map[string]bool, out *[]string) {
	for _, term := range splitIdentifierTerms(text) {
		if len(term) < 3 || seen[term] {
			continue
		}
		seen[term] = true
		*out = append(*out, term)
	}
}

func splitIdentifierTerms(text string) []string {
	fields := strings.FieldsFunc(text, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
	})
	var out []string
	for _, field := range fields {
		field = strings.ToLower(strings.TrimSpace(field))
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}
