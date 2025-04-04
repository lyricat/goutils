package core

import (
	"context"
	"fmt"
)

type (
	ChainParamsStep struct {
		Input       string
		Instruction string
		Options     any
	}

	ChainParams struct {
		Format           string
		Steps            []ChainParamsStep
		RawRequestParams map[string]any
	}

	Config struct {
		// openai
		OpenAIAPIBase        string
		OpenAIAPIKey         string
		OpenAIModel          string
		OpenAIEmbeddingModel string

		// azure openai
		AzureOpenAIAPIKey         string
		AzureOpenAIEndpoint       string
		AzureOpenAIModel          string
		AzureOpenAIEmbeddingModel string

		// aws bedrock
		AwsKey                      string
		AwsSecret                   string
		AwsBedrockModelArn          string
		AwsBedrockEmbeddingModelArn string

		// susanoo
		SusanooAPIBase string
		SusanooAPIKey  string

		// gemini
		GeminiAPIKey string

		Provider string

		Debug bool
	}

	GeneralChatCompletionMessage struct {
		Role        string `json:"role"`
		Content     string `json:"content"`
		EnableCache bool   `json:"enable_cache,omitempty"`
	}

	ResultUsage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		CachedTokens int `json:"cached_tokens"`
	}

	Result struct {
		Text  string         `json:"text"`
		Json  map[string]any `json:"json"`
		Usage ResultUsage    `json:"usage"`
	}

	AIInstant interface {
		RawRequest(ctx context.Context, messages []GeneralChatCompletionMessage) (*Result, error)
		RawRequestWithParams(ctx context.Context, messages []GeneralChatCompletionMessage, params map[string]any) (*Result, error)
		OneTimeRequestWithParams(ctx context.Context, content string, params map[string]any) (*Result, error)
	}
)

const (
	ProviderAzure    = "azure"
	ProviderOpenAI   = "openai"
	ProviderBedrock  = "bedrock"
	ProviderSusanoo  = "susanoo"
	ProviderDeepseek = "deepseek"
	ProviderXAI      = "xai"
)

func (m GeneralChatCompletionMessage) Pretty() string {
	return fmt.Sprintf("{ Role: '%s', Content: '%s' }", m.Role, m.Content)
}
