package ai

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
)

func (s *Instant) RawRequestAzureOpenAI(ctx context.Context, messages []azopenai.ChatRequestMessageClassification) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	resultChan := make(chan struct {
		resp string
		err  error
	})

	go func() {

		resp, err := s.azureOpenAIClient.GetChatCompletions(ctx, azopenai.ChatCompletionsOptions{
			Messages:       messages,
			DeploymentName: &s.cfg.AzureOpenAIGptDeploymentID,
		}, nil)

		if err != nil {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: err}
			return
		}

		if len(resp.Choices) == 0 {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: nil}
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
				if *choice.ContentFilterResults.Hate.Filtered {
					err = fmt.Errorf("content filter hate: %v", *choice.ContentFilterResults.Hate.Severity)
				}
				if *choice.ContentFilterResults.SelfHarm.Filtered {
					err = fmt.Errorf("content filter self harm: %v", *choice.ContentFilterResults.SelfHarm.Severity)
				}
				if *choice.ContentFilterResults.Sexual.Filtered {
					err = fmt.Errorf("content filter sexual: %v", *choice.ContentFilterResults.Sexual.Severity)
				}
				if *choice.ContentFilterResults.Violence.Filtered {
					err = fmt.Errorf("content filter violence: %v", *choice.ContentFilterResults.Violence.Severity)
				}
				if err != nil {
					resultChan <- struct {
						resp string
						err  error
					}{resp: "", err: err}
					return
				}
			}
		}

		if !gotReply {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: nil}
			return
		}

		ret := resp.Choices[0].Message.Content
		resultChan <- struct {
			resp string
			err  error
		}{resp: *ret, err: nil}
	}()

	select {
	case <-ctx.Done():
		// Context was canceled or timed out
		if errors.Is(ctx.Err(), context.Canceled) {
			slog.Error("[common.ai] Azure Request canceled", "error", ctx.Err())
			return "", fmt.Errorf("request canceled: %w", ctx.Err())
		}
		return "", fmt.Errorf("request failed: %w", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, context.Canceled) {
				slog.Error("[common.ai] Azure Request canceled", "error", result.err)
				return "", fmt.Errorf("request canceled: %w", result.err)
			}
			slog.Error("[common.ai] Azure Request error", "error", result.err)
			return "", result.err
		}
		return result.resp, nil
	}
}

func (s *Instant) CreateEmbeddingAzureOpenAI(ctx context.Context, input []string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	resp, err := s.azureOpenAIClient.GetEmbeddings(ctx, azopenai.EmbeddingsOptions{
		Input:          input,
		DeploymentName: &s.cfg.AzureOpenAIEmbeddingDeploymentID,
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
