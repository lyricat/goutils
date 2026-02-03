package embedding

import (
	"context"

	aiembedding "github.com/lyricat/goutils/ai/embedding"
)

type Config struct {
	JinaAPIKey    string
	JinaAPIBase   string
	OpenAIAPIKey  string
	OpenAIAPIBase string
	GeminiAPIKey  string
	GeminiAPIBase string
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Create(ctx context.Context, opts ...Option) (*Result, error) {
	req := BuildRequest(opts...)
	input := &aiembedding.CreateEmbeddingsInput{
		Provider:      req.Provider,
		Model:         req.Model,
		Input:         toLegacyInputs(req.Input),
		JinaOptions:   req.Options.Jina,
		OpenAIOptions: req.Options.OpenAI,
		GeminiOptions: req.Options.Gemini,
	}

	legacy := aiembedding.NewEmbedding(&aiembedding.Config{
		JinaAPIKey:    c.cfg.JinaAPIKey,
		JinaAPIBase:   c.cfg.JinaAPIBase,
		OpenAIAPIKey:  c.cfg.OpenAIAPIKey,
		OpenAIAPIBase: c.cfg.OpenAIAPIBase,
		GeminiAPIKey:  c.cfg.GeminiAPIKey,
		GeminiAPIBase: c.cfg.GeminiAPIBase,
	})
	return legacy.CreateEmbeddings(ctx, input)
}

func toLegacyInputs(inputs []Input) []aiembedding.CreateEmbeddingsInputItem {
	out := make([]aiembedding.CreateEmbeddingsInputItem, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, aiembedding.CreateEmbeddingsInputItem{
			Text:  in.Text,
			Image: in.Image,
		})
	}
	return out
}
