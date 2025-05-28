package core

import (
	"context"
	"fmt"
)

const (
	FormatJSON = "json"
	FormatYAML = "yaml"
)

const (
	JINA_API_BASE = "https://api.jina.ai/v1"
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

		// anthropic
		AnthropicAPIKey string
		AnthropicModel  string

		// susanoo
		SusanooAPIBase string
		SusanooAPIKey  string

		Provider string

		Debug bool
	}

	RawRequestOptions struct {
		Format string
		Model  string
	}

	Message struct {
		Role        string `json:"role"`
		Content     string `json:"content"`
		EnableCache bool   `json:"enable_cache,omitempty"`
	}

	ResultUsage struct {
		InputTokens              int `json:"input_tokens"`
		OutputTokens             int `json:"output_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheInputTokens         int `json:"cache_input_tokens"`
	}

	Result struct {
		Text  string         `json:"text"`
		Json  map[string]any `json:"json"`
		Usage ResultUsage    `json:"usage"`
	}

	AIInstant interface {
		RawRequest(ctx context.Context, messages []Message) (*Result, error)
		RawRequestWithParams(ctx context.Context, messages []Message, params map[string]any) (*Result, error)
		OneTimeRequestWithParams(ctx context.Context, content string, params map[string]any) (*Result, error)
	}
)

const (
	ProviderOpenAI = "openai"
	// openai compatible
	ProviderOpenAICustom = "openai_custom"
	ProviderDeepseek     = "deepseek"
	ProviderXAI          = "xai"
	ProviderGemini       = "gemini"
	// azure openai
	ProviderAzure = "azure"
	// anthropic
	ProviderAnthropic = "anthropic"
	ProviderBedrock   = "bedrock"
	// jina
	ProviderJina = "jina"
	// susanoo
	ProviderSusanoo = "susanoo"
)

func (m Message) Pretty() string {
	return fmt.Sprintf("{ Role: '%s', Content: '%s' }", m.Role, m.Content)
}
