package classify

import aiclassify "github.com/lyricat/goutils/ai/classify"

type Input struct {
	Text  string `json:"text,omitempty"`
	Image string `json:"image,omitempty"`
}

type Request struct {
	Provider string   `json:"provider,omitempty"`
	Model    string   `json:"model,omitempty"`
	Input    []Input  `json:"input,omitempty"`
	Labels   []string `json:"labels,omitempty"`
}

type Result = aiclassify.ClassifyOutput

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

func Classify(model string, labels []string, inputs ...Input) Option {
	return func(r *Request) {
		r.Model = model
		r.Labels = append([]string{}, labels...)
		r.Input = append(r.Input, inputs...)
	}
}

func WithProvider(provider string) Option {
	return func(r *Request) { r.Provider = provider }
}
