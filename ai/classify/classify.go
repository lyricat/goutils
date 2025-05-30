package rerank

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lyricat/goutils/structs"
)

type (
	ClassifyClient struct {
		cfg *Config
	}
	Config struct {
		JinaAPIKey  string
		JinaAPIBase string
	}

	ClassifyInput struct {
		Provider string              `json:"provider"`
		Model    string              `json:"model"`
		Input    []ClassifyInputItem `json:"input"`
		Labels   []string            `json:"labels"`
	}

	ClassifyInputItem struct {
		Text  string `json:"text,omitempty"`
		Image string `json:"image,omitempty"`
	}

	ClassifyOutput struct {
		Data []struct {
			Object      string  `json:"object"`
			Index       int     `json:"index"`
			Prediction  string  `json:"prediction"`
			Score       float64 `json:"score"`
			Predictions []struct {
				Label string  `json:"label"`
				Score float64 `json:"score"`
			} `json:"predictions"`
		} `json:"data"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
)

func (i *ClassifyInput) ToJSONMap() (structs.JSONMap, error) {
	js, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return structs.NewFromJSONString(string(js)), nil
}

func NewClassifyClient(cfg *Config) *ClassifyClient {
	return &ClassifyClient{
		cfg: cfg,
	}
}

func (c *ClassifyClient) Classify(ctx context.Context, input *ClassifyInput) (*ClassifyOutput, error) {
	// if provider is not set
	// look at the model name, and decide which provider to use
	// else, use the provider in the input
	if input.Provider == "" {
		input.Provider = pickProviderByModel(input.Model)
	}

	if input.Provider == "" {
		return nil, fmt.Errorf("provider not set")
	}

	var resp *ClassifyOutput
	var err error
	switch input.Provider {
	case "jina":
		resp, err = JinaClassify(ctx, c.cfg.JinaAPIKey, c.cfg.JinaAPIBase, input)
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
	return ""
}
