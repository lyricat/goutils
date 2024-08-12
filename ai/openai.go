package ai

import (
	"context"
	"log/slog"
	"time"

	openai "github.com/sashabaranov/go-openai"
)

func (s *Instant) RawRequestOpenAI(ctx context.Context, messages []openai.ChatCompletionMessage) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	resp, err := s.openaiClient.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model:    openai.GPT3Dot5Turbo1106,
			Messages: messages,
		},
	)

	if err != nil {
		slog.Error("[common.ai] OpenAI OneTimeRequest error", "error", err)
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", nil
	}

	return resp.Choices[0].Message.Content, nil
}
