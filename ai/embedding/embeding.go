package embedding

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lyricat/goutils/structs"
)

type (
	EmbeddingClient struct {
		cfg *Config
	}
	Config struct {
		JinaAPIKey    string
		JinaAPIBase   string
		OpenAIAPIKey  string
		OpenAIAPIBase string
	}

	CreateEmbeddingsInputItem struct {
		Text  string `json:"text,omitempty"`
		Image string `json:"image,omitempty"`
	}

	CreateEmbeddingsInput struct {
		Provider      string                      `json:"provider"`
		Model         string                      `json:"model"`
		Input         []CreateEmbeddingsInputItem `json:"input"`
		JinaOptions   structs.JSONMap             `json:"jina_options"`
		OpenAIOptions structs.JSONMap             `json:"openai_options"`
	}

	CreateEmbeddingsOutput struct {
		Model  string `json:"model"`
		Object string `json:"object"`
		Data   []struct {
			Object    string `json:"object"`
			Embedding string `json:"embedding"`
			Index     int    `json:"index"`
		} `json:"data"`
		Usage struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}
)

func (i *CreateEmbeddingsInput) ToJSONMap() (structs.JSONMap, error) {
	js, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return structs.NewFromJSONString(string(js)), nil
}

func NewEmbedding(cfg *Config) *EmbeddingClient {
	return &EmbeddingClient{
		cfg: cfg,
	}
}

func (c *EmbeddingClient) CreateEmbeddings(ctx context.Context, input *CreateEmbeddingsInput) (*CreateEmbeddingsOutput, error) {
	// if provider is not set
	// look at the model name, and decide which provider to use
	// else, use the provider in the input
	if input.Provider == "" {
		input.Provider = pickProviderByModel(input.Model)
	}

	if input.Provider == "" {
		return nil, fmt.Errorf("provider not set")
	}

	var resp *CreateEmbeddingsOutput
	var err error
	switch input.Provider {
	case "jina":
		resp, err = JinaCreateEmbeddings(ctx, c.cfg.JinaAPIKey, c.cfg.JinaAPIBase, input)
	case "openai":
		resp, err = OpenAICreateEmbeddings(ctx, c.cfg.OpenAIAPIKey, c.cfg.OpenAIAPIBase, input)
	default:
		return nil, fmt.Errorf("unknown provider: %s", input.Provider)
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func pickProviderByModel(model string) string {
	if strings.Contains(model, "jina") {
		return "jina"
	}
	return "openai"
}
