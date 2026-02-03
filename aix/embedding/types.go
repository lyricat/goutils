package embedding

import (
	aiembedding "github.com/lyricat/goutils/ai/embedding"
	"github.com/lyricat/goutils/structs"
)

type Input struct {
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"`
}

type Options struct {
	Jina   structs.JSONMap `json:"jina_options,omitempty"`
	OpenAI structs.JSONMap `json:"openai_options,omitempty"`
	Gemini structs.JSONMap `json:"gemini_options,omitempty"`
}

type Request struct {
	Provider string  `json:"provider,omitempty"`
	Model    string  `json:"model,omitempty"`
	Input    []Input `json:"input"`
	Options  Options `json:"options,omitempty"`
}

type Result = aiembedding.CreateEmbeddingsOutput

type Option func(*Request)

func BuildRequest(opts ...Option) *Request {
	req := &Request{}
	for _, opt := range opts {
		if opt != nil {
			opt(req)
		}
	}
	return req
}

func Embedding(model string, texts ...string) Option {
	return func(r *Request) {
		r.Model = model
		for _, t := range texts {
			r.Input = append(r.Input, Input{Text: t})
		}
	}
}

func WithProvider(provider string) Option {
	return func(r *Request) { r.Provider = provider }
}

func WithInputs(inputs ...Input) Option {
	return func(r *Request) { r.Input = append(r.Input, inputs...) }
}

func WithOptions(opts Options) Option {
	return func(r *Request) { r.Options = opts }
}
