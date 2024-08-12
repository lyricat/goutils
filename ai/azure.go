package ai

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/ai/azopenai"
)

func (s *Instant) RawRequestAzureOpenAI(ctx context.Context, messages []azopenai.ChatRequestMessageClassification) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	resp, err := s.azureOpenAIClient.GetChatCompletions(ctx, azopenai.ChatCompletionsOptions{
		Messages:       messages,
		DeploymentName: &s.cfg.AzureOpenAIGptDeploymentID,
	}, nil)

	if err != nil {
		slog.Error("[common.azure] RawRequestAzure error", "error", err)
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", nil
	}

	gotReply := false
	for _, choice := range resp.Choices {
		gotReply = true
		if choice.ContentFilterResults != nil {
			if choice.ContentFilterResults.Error != nil {
				return "", fmt.Errorf("content filter error: %v", *choice.ContentFilterResults.Error)
			}
			if *choice.ContentFilterResults.Hate.Filtered {
				return "", fmt.Errorf("content filter hate: %v", *choice.ContentFilterResults.Hate.Severity)
			}
			if *choice.ContentFilterResults.SelfHarm.Filtered {
				return "", fmt.Errorf("content filter self harm: %v", *choice.ContentFilterResults.SelfHarm.Severity)
			}
			if *choice.ContentFilterResults.Sexual.Filtered {
				return "", fmt.Errorf("content filter sexual: %v", *choice.ContentFilterResults.Sexual.Severity)
			}
			if *choice.ContentFilterResults.Violence.Filtered {
				return "", fmt.Errorf("content filter violence: %v", *choice.ContentFilterResults.Violence.Severity)
			}
		}
	}

	if !gotReply {
		return "", nil
	}

	ret := resp.Choices[0].Message.Content
	return *ret, nil
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
