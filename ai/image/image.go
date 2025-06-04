package image

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lyricat/goutils/structs"
)

type (
	Config struct {
		OpenAIAPIKey  string
		OpenAIAPIBase string
		GeminiAPIKey  string
	}

	ImageClient struct {
		cfg *Config
	}

	CreateImagesInput struct {
		Provider      string          `json:"provider"`
		Model         string          `json:"model"`
		Prompt        string          `json:"prompt"`
		Count         int             `json:"count"`
		OpenAIOptions structs.JSONMap `json:"openai_options"`
		GeminiOptions structs.JSONMap `json:"gemini_options"`
	}

	CreateImagesOutput struct {
		Created int `json:"created"`
		Data    []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
		MimeType string           `json:"mime_type"`
		Usage    CreateImageUsage `json:"usage"`
	}

	CreateImageUsage struct {
		Size    string `json:"size"`
		Quality string `json:"quality"`

		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
		TotalTokens  int `json:"total_tokens"`
	}
)

func (m *CreateImagesInput) ToJSONMap() (structs.JSONMap, error) {
	js, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return structs.NewFromJSONString(string(js)), nil
}

func NewImageClient(cfg *Config) *ImageClient {
	return &ImageClient{
		cfg: cfg,
	}
}

func (c *ImageClient) CreateImages(ctx context.Context, input *CreateImagesInput) (*CreateImagesOutput, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*600)
	defer cancel()

	// if provider is not set
	// look at the model name, and decide which provider to use
	// else, use the provider in the input
	if input.Provider == "" {
		input.Provider = pickProviderByModel(input.Model)
	}

	if input.Provider == "" {
		return nil, fmt.Errorf("provider not set")
	}

	var resp *CreateImagesOutput
	var err error
	switch input.Provider {
	case "openai", "openai_custom":
		resp, err = OpenAICreateImages(ctx, c.cfg.OpenAIAPIKey, c.cfg.OpenAIAPIBase, input)
	case "gemini":
		resp, err = GeminiCreateImages(ctx, c.cfg.GeminiAPIKey, input)
	default:
		return nil, fmt.Errorf("unknown provider: %s", input.Provider)
	}
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func pickProviderByModel(model string) string {
	if strings.Contains(model, "gpt-") {
		return "openai"
	}
	return "openai"
}
