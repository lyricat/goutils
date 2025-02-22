package ai

import (
	"errors"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func supportJSONResponse(model string) bool {
	return strings.HasPrefix(model, "gpt-4") || strings.HasPrefix(model, "gpt-3.5") ||
		strings.HasPrefix(model, "deepseek-chat") || strings.HasPrefix(model, "grok-")
}

func isOpenAICompatible(cfg Config) bool {
	compatibleProviders := []string{"openai", "deepseek", "xai"}
	for _, provider := range compatibleProviders {
		if cfg.Provider == provider {
			return true
		}
	}
	return false
}

func createOpenAICompatibleClient(cfg Config) (*openai.Client, error) {
	config := openai.DefaultConfig(cfg.OpenAIAPIKey)
	switch cfg.Provider {
	case "openai":
		// no-op
	case "deepseek":
		config.BaseURL = "https://api.deepseek.com"
	case "xai":
		config.BaseURL = "https://api.x.ai/v1"
	default:
		return nil, errors.New("unsupported provider")
	}

	return openai.NewClientWithConfig(config), nil
}
