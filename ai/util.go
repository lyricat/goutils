package ai

import (
	"errors"
	"strings"

	"github.com/lyricat/goutils/ai/core"
	"github.com/sashabaranov/go-openai"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func supportJSONResponse(model string) bool {
	return strings.HasPrefix(model, "gpt-4") || strings.HasPrefix(model, "gpt-3.5") ||
		strings.HasPrefix(model, "deepseek-chat") || strings.HasPrefix(model, "grok-")
}

func IsOpenAICompatible(p string) bool {
	compatibleProviders := []string{"openai", "deepseek", "xai", "gemini"}
	for _, provider := range compatibleProviders {
		if p == provider {
			return true
		}
	}
	return false
}

func createOpenAICompatibleClient(cfg core.Config) (*openai.Client, error) {
	config := openai.DefaultConfig(cfg.OpenAIAPIKey)
	switch cfg.Provider {
	case core.ProviderOpenAI:
		// pass
	case core.ProviderDeepseek:
		config.BaseURL = "https://api.deepseek.com"
	case core.ProviderXAI:
		config.BaseURL = "https://api.x.ai/v1"
	case core.ProviderGemini:
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta/openai"
	default:
		return nil, errors.New("unsupported provider")
	}

	return openai.NewClientWithConfig(config), nil
}

func extractJSONFromMarkdown(markdownContent string) (string, error) {
	md := goldmark.New(
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	// Parse the markdown content
	reader := text.NewReader([]byte(markdownContent))
	doc := md.Parser().Parse(reader)

	jsonContents := make([]string, 0)
	// Traverse the AST to find JSON code blocks
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Check if the node is a fenced code block
		if codeBlock, ok := n.(*ast.FencedCodeBlock); ok {
			// Get the language info
			lang := string(codeBlock.Language(reader.Source()))
			if lang == "json" {
				// Extract the content inside the code block
				content := codeBlock.Text(reader.Source())
				// Convert to string
				jsonContent := string(content)
				// Append to the list of JSON contents
				jsonContents = append(jsonContents, jsonContent)
				// Continue walking to find more JSON blocks
			}
		}

		return ast.WalkContinue, nil
	})

	return strings.Join(jsonContents, "\n"), nil
}
