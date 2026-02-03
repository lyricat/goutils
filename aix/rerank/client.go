package rerank

import (
	"context"

	airerank "github.com/lyricat/goutils/ai/rerank"
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

func (c *Client) Rerank(ctx context.Context, opts ...Option) (*Result, error) {
	req := BuildRequest(opts...)
	input := &airerank.RerankInput{
		Provider:        req.Provider,
		Model:           req.Model,
		Query:           req.Query,
		TopN:            req.TopN,
		ReturnDocuments: req.ReturnDocuments,
		Documents:       toLegacyDocs(req.Documents),
	}
	legacy := airerank.NewRerank(&airerank.Config{
		JinaAPIKey:  c.cfg.JinaAPIKey,
		JinaAPIBase: c.cfg.JinaAPIBase,
	})
	return legacy.Rerank(ctx, input)
}

func toLegacyDocs(docs []Input) []airerank.RerankInputDocItem {
	out := make([]airerank.RerankInputDocItem, 0, len(docs))
	for _, d := range docs {
		out = append(out, airerank.RerankInputDocItem{
			Text:  d.Text,
			Image: d.Image,
		})
	}
	return out
}
