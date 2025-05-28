package rerank

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lyricat/goutils/structs"
)

type (
	RerankClient struct {
		cfg *Config
	}
	Config struct {
		JinaAPIKey  string
		JinaAPIBase string
	}

	RerankInput struct {
		Provider        string               `json:"provider"`
		Model           string               `json:"model"`
		Query           string               `json:"query"`
		Documents       []RerankInputDocItem `json:"documents"`
		TopN            int                  `json:"top_n"`
		ReturnDocuments bool                 `json:"return_documents"`
	}

	RerankInputDocItem struct {
		Text  string `json:"text,omitempty"`
		Image string `json:"image,omitempty"`
	}

	RerankOutput struct {
		Model   string `json:"model"`
		Results []struct {
			RelevanceScore float64 `json:"relevance_score"`
			Index          int     `json:"index"`
			Document       struct {
				Text string `json:"text,omitempty"`
				URL  string `json:"url,omitempty"`
			} `json:"document,omitempty"`
		} `json:"results"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
)

func (i *RerankInput) ToJSONMap() (structs.JSONMap, error) {
	js, err := json.Marshal(i)
	if err != nil {
		return nil, err
	}
	return structs.NewFromJSONString(string(js)), nil
}

func NewRerank(cfg *Config) *RerankClient {
	return &RerankClient{
		cfg: cfg,
	}
}

func (c *RerankClient) Rerank(ctx context.Context, input *RerankInput) (*RerankOutput, error) {
	// if provider is not set
	// look at the model name, and decide which provider to use
	// else, use the provider in the input
	if input.Provider == "" {
		input.Provider = pickProviderByModel(input.Model)
	}

	if input.Provider == "" {
		return nil, fmt.Errorf("provider not set")
	}

	var resp *RerankOutput
	var err error
	switch input.Provider {
	case "jina":
		resp, err = JinaRerank(ctx, c.cfg.JinaAPIKey, c.cfg.JinaAPIBase, input)
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
