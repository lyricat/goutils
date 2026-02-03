package classify

import (
	"context"

	aiclassify "github.com/lyricat/goutils/ai/classify"
)

type Config struct {
	JinaAPIKey  string
	JinaAPIBase string
}

type Client struct {
	cfg Config
}

func New(cfg Config) *Client {
	return &Client{cfg: cfg}
}

func (c *Client) Classify(ctx context.Context, opts ...Option) (*Result, error) {
	req := BuildRequest(opts...)
	input := &aiclassify.ClassifyInput{
		Provider: req.Provider,
		Model:    req.Model,
		Labels:   req.Labels,
		Input:    toLegacyInputs(req.Input),
	}
	legacy := aiclassify.NewClassifyClient(&aiclassify.Config{
		JinaAPIKey:  c.cfg.JinaAPIKey,
		JinaAPIBase: c.cfg.JinaAPIBase,
	})
	return legacy.Classify(ctx, input)
}

func toLegacyInputs(inputs []Input) []aiclassify.ClassifyInputItem {
	out := make([]aiclassify.ClassifyInputItem, 0, len(inputs))
	for _, in := range inputs {
		out = append(out, aiclassify.ClassifyInputItem{
			Text:  in.Text,
			Image: in.Image,
		})
	}
	return out
}
