package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/lyricat/goutils/ai/core"
	openai "github.com/sashabaranov/go-openai"
)

type (
	OpenAIRawRequestOptions struct {
		UseJSON bool
	}
)

func (s *Instant) OpenAIRawRequest(ctx context.Context, messages []openai.ChatCompletionMessage, opts *OpenAIRawRequestOptions) (*core.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	resultChan := make(chan struct {
		resp *core.Result
		err  error
	})

	go func() {
		payload := openai.ChatCompletionRequest{
			Model:    s.cfg.OpenAIModel,
			Messages: messages,
		}

		if opts != nil {
			if opts.UseJSON && supportJSONResponse(s.cfg.OpenAIModel) {
				payload.ResponseFormat = &openai.ChatCompletionResponseFormat{
					Type: openai.ChatCompletionResponseFormatTypeJSONObject,
				}
			}
		}

		resp, err := s.openaiClient.CreateChatCompletion(ctx, payload)
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

		r := &core.Result{Text: resp.Choices[0].Message.Content}
		r.Usage.InputTokens = resp.Usage.PromptTokens
		r.Usage.OutputTokens = resp.Usage.CompletionTokens
		if resp.Usage.PromptTokensDetails != nil {
			r.Usage.CacheInputTokens = resp.Usage.PromptTokensDetails.CachedTokens
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
			slog.Error("[goutils.ai] OpenAI Request canceled", "error", ctx.Err())
			return nil, fmt.Errorf("request canceled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("request failed: %w", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, context.Canceled) {
				slog.Error("[goutils.ai] OpenAI Request canceled", "error", result.err)
				return nil, fmt.Errorf("request canceled: %w", result.err)
			}
			slog.Error("[goutils.ai] OpenAI Request error", "error", result.err)
			return nil, result.err
		}
		return result.resp, nil
	}
}

func (s *Instant) CreateEmbeddingOpenAI(ctx context.Context, input []string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	resp, err := s.openaiClient.CreateEmbeddings(ctx, openai.EmbeddingRequest{
		Input: input,
		Model: openai.EmbeddingModel(s.cfg.OpenAIEmbeddingModel),
	})
	if err != nil {
		slog.Error("[goutils.ai] CreateEmbeddingOpenAI error", "error", err)
		return nil, err
	}

	if len(resp.Data) > 0 {
		// Convert []float64 to []float32 to match Azure format
		embeddings := make([]float32, len(resp.Data[0].Embedding))
		for i, v := range resp.Data[0].Embedding {
			embeddings[i] = float32(v)
		}
		return embeddings, nil
	}

	return nil, nil
}
