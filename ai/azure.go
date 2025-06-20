package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
	"github.com/lyricat/goutils/ai/core"
)

func (s *Instant) AzureOpenAIRawRequest(ctx context.Context, messages []azopenai.ChatRequestMessageClassification, opts *core.RawRequestOptions) (*core.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*600)
	defer cancel()

	resultChan := make(chan struct {
		resp *core.Result
		err  error
	})

	go func() {

		payload := azopenai.ChatCompletionsOptions{
			Messages:       messages,
			DeploymentName: &s.cfg.AzureOpenAIModel,
		}

		if opts != nil {
			if opts.Format == core.FormatJSON && supportJSONResponse(s.cfg.AzureOpenAIModel) {
				payload.ResponseFormat = &azopenai.ChatCompletionsJSONResponseFormat{}
			}
		}

		resp, err := s.azureOpenAIClient.GetChatCompletions(ctx, payload, nil)

		if err != nil {
			resultChan <- struct {
				resp *core.Result
				err  error
			}{resp: nil, err: err}
			return
		}

		if len(resp.Choices) == 0 {
			resultChan <- struct {
				resp *core.Result
				err  error
			}{resp: nil, err: nil}
			return
		}

		gotReply := false
		for _, choice := range resp.Choices {
			gotReply = true
			if choice.ContentFilterResults != nil {
				var err error
				if choice.ContentFilterResults.Error != nil {
					err = fmt.Errorf("content filter error: %v", *choice.ContentFilterResults.Error)
				}
				if choice.ContentFilterResults != nil {
					if choice.ContentFilterResults.Hate != nil {
						if *(choice.ContentFilterResults.Hate.Filtered) {
							err = fmt.Errorf("content filter hate: %v", *choice.ContentFilterResults.Hate.Severity)
						}
					}
					if choice.ContentFilterResults.SelfHarm != nil {
						if *choice.ContentFilterResults.SelfHarm.Filtered {
							err = fmt.Errorf("content filter self harm: %v", *choice.ContentFilterResults.SelfHarm.Severity)
						}
					}
					if choice.ContentFilterResults.Sexual != nil {
						if *choice.ContentFilterResults.Sexual.Filtered {
							err = fmt.Errorf("content filter sexual: %v", *choice.ContentFilterResults.Sexual.Severity)
						}
					}
					if choice.ContentFilterResults.Violence != nil {
						if *choice.ContentFilterResults.Violence.Filtered {
							err = fmt.Errorf("content filter violence: %v", *choice.ContentFilterResults.Violence.Severity)
						}
					}
					if err != nil {
						resultChan <- struct {
							resp *core.Result
							err  error
						}{resp: nil, err: err}
						return
					}
				}
			}
		}

		if !gotReply {
			resultChan <- struct {
				resp *core.Result
				err  error
			}{resp: nil, err: nil}
			return
		}

		r := &core.Result{Text: *resp.Choices[0].Message.Content}
		r.Usage.InputTokens = int(*resp.Usage.PromptTokens)
		r.Usage.OutputTokens = int(*resp.Usage.CompletionTokens)
		if resp.Usage.PromptTokensDetails != nil {
			r.Usage.CacheInputTokens = int(*resp.Usage.PromptTokensDetails.CachedTokens)
		}

		resultChan <- struct {
			resp *core.Result
			err  error
		}{resp: r, err: nil}
	}()

	select {
	case <-ctx.Done():
		// Context was canceled or timed out
		if errors.Is(ctx.Err(), context.Canceled) {
			slog.Error("[goutils.ai] Azure Request canceled", "error", ctx.Err())
			return nil, fmt.Errorf("request canceled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("request failed: %w", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, context.Canceled) {
				slog.Error("[goutils.ai] Azure Request canceled", "error", result.err)
				return nil, fmt.Errorf("request canceled: %w", result.err)
			}
			slog.Error("[goutils.ai] Azure Request error", "error", result.err)
			return nil, result.err
		}
		return result.resp, nil
	}
}

func (s *Instant) CreateEmbeddingAzureOpenAI(ctx context.Context, input []string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	resp, err := s.azureOpenAIClient.GetEmbeddings(ctx, azopenai.EmbeddingsOptions{
		Input:          input,
		DeploymentName: &s.cfg.AzureOpenAIEmbeddingModel,
	}, nil)

	if err != nil {
		slog.Error("[common.azure] CreateEmbeddingAzure error", "error", err)
		return nil, err
	}

	if len(resp.Embeddings.Data) != 0 {
		return resp.Embeddings.Data[0].Embedding, nil
	}

	return nil, nil
}
