package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/bedrockruntime"
)

type (
	BedRockClaudeChatMessage struct {
		Role    string                        `json:"role"`
		Content []BedRockClaudeMessageContent `json:"content"`
	}

	BedRockClaudeMessageContent struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	}

	BedrockClaudeResponse struct {
		ID           string                        `json:"id"`
		Type         string                        `json:"type"`
		Role         string                        `json:"role"`
		Model        string                        `json:"model"`
		Content      []BedRockClaudeMessageContent `json:"content"`
		StopReason   string                        `json:"stop_reason"`
		StopSequence interface{}                   `json:"stop_sequence"`
		Usage        map[string]int                `json:"usage"`
	}
)

const (
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
)

func (s *Instant) RawRequestAWSBedrockClaude(ctx context.Context, messages []BedRockClaudeChatMessage) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	resultChan := make(chan struct {
		resp string
		err  error
	})

	go func() {
		body := map[string]interface{}{
			"anthropic_version": "bedrock-2023-05-31",
			"max_tokens":        10000,
			"messages":          messages,
		}

		bodyBytes, err := json.Marshal(body)
		if err != nil {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: fmt.Errorf("failed to marshal request body: %w", err)}
			return
		}

		resp, err := s.bedrockClient.InvokeModelWithContext(ctx, &bedrockruntime.InvokeModelInput{
			ModelId:     aws.String(s.cfg.AwsBedrockModelArn),
			Body:        []byte(bodyBytes),
			Accept:      aws.String("application/json"),
			ContentType: aws.String("application/json"),
		})

		if err != nil {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: err}
			return
		}

		var r BedrockClaudeResponse
		if err := json.Unmarshal([]byte(resp.Body), &r); err != nil {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: fmt.Errorf("failed to unmarshal response: %w", err)}
			return
		}

		if len(r.Content) == 0 {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: nil}
			return
		}

		resultChan <- struct {
			resp string
			err  error
		}{resp: r.Content[0].Text, err: nil}
	}()

	select {
	case <-ctx.Done():
		// Context was canceled or timed out
		if errors.Is(ctx.Err(), context.Canceled) {
			slog.Error("[goutils.ai] AWS Bedrock Request canceled", "error", ctx.Err())
			return "", fmt.Errorf("request canceled: %w", ctx.Err())
		}
		return "", fmt.Errorf("request failed: %w", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, context.Canceled) {
				slog.Error("[goutils.ai] AWS Bedrock Request canceled", "error", result.err)
				return "", fmt.Errorf("request canceled: %w", result.err)
			}
			slog.Error("[goutils.ai] AWS Bedrock Request error", "error", result.err)
			return "", result.err
		}
		return result.resp, nil
	}
}
