package image

import (
	"context"

	aiimage "github.com/lyricat/goutils/ai/image"
)

type Config struct {
	OpenAIAPIKey  string
	OpenAIAPIBase string
	GeminiAPIKey  string
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Create(ctx context.Context, opts ...Option) (*Result, error) {
	req := BuildRequest(opts...)

	input := &aiimage.CreateImagesInput{
		Provider:      req.Provider,
		Model:         req.Model,
		Prompt:        req.Prompt,
		Count:         req.Count,
		OpenAIOptions: req.Options.OpenAI,
		GeminiOptions: req.Options.Gemini,
	}

	legacy := aiimage.NewImageClient(&aiimage.Config{
		OpenAIAPIKey:  c.cfg.OpenAIAPIKey,
		OpenAIAPIBase: c.cfg.OpenAIAPIBase,
		GeminiAPIKey:  c.cfg.GeminiAPIKey,
	})

	return legacy.CreateImages(ctx, input)
}
