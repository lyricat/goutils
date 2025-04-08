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
	"github.com/lyricat/goutils/ai/core"
)

type (
	// Claude's Input & Output
	// FYI: https://docs.aws.amazon.com/ja_jp/bedrock/latest/userguide/model-parameters-anthropic-claude-messages.html
	BedRockClaudeChatMessage struct {
		Role    string                        `json:"role"`
		Content []BedRockClaudeMessageContent `json:"content"`
	}

	BedRockClaudeCacheControl struct {
		Type string `json:"type"` // default, "ephemeral"
	}

	BedRockClaudeMessageContent struct {
		Type         string                     `json:"type"`
		Text         string                     `json:"text,omitempty"`
		CacheControl *BedRockClaudeCacheControl `json:"cache_control,omitempty"`
	}

	BedrockClaudeResponse struct {
		ID           string                        `json:"id"`
		Type         string                        `json:"type"`
		Role         string                        `json:"role"`
		Model        string                        `json:"model"`
		Content      []BedRockClaudeMessageContent `json:"content"`
		StopReason   string                        `json:"stop_reason"`
		StopSequence interface{}                   `json:"stop_sequence"`
		Usage        struct {
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
		} `json:"usage"`
	}
)

const (
	ChatMessageRoleUser      = "user"
	ChatMessageRoleAssistant = "assistant"
)

func (s *Instant) BedrockRawRequest(ctx context.Context, messages []BedRockClaudeChatMessage, _opts *core.RawRequestOptions) (*core.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	resultChan := make(chan struct {
		resp *core.Result
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
				resp *core.Result
				err  error
			}{resp: nil, err: fmt.Errorf("failed to marshal request body: %w", err)}
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
				resp *core.Result
				err  error
			}{resp: nil, err: err}
			return
		}

		var r BedrockClaudeResponse
		if err := json.Unmarshal([]byte(resp.Body), &r); err != nil {
			resultChan <- struct {
				resp *core.Result
				err  error
			}{resp: nil, err: fmt.Errorf("failed to unmarshal response: %w", err)}
			return
		}

		if len(r.Content) == 0 {
			resultChan <- struct {
				resp *core.Result
				err  error
			}{resp: nil, err: nil}
			return
		}

		result := &core.Result{
			Text: r.Content[0].Text,
		}
		result.Usage.InputTokens = r.Usage.InputTokens
		result.Usage.OutputTokens = r.Usage.OutputTokens

		result.Usage.CacheInputTokens = r.Usage.CacheCreationInputTokens
		result.Usage.CacheReadInputTokens = r.Usage.CacheReadInputTokens

		resultChan <- struct {
			resp *core.Result
			err  error
		}{resp: result, err: nil}
	}()

	select {
	case <-ctx.Done():
		// Context was canceled or timed out
		if errors.Is(ctx.Err(), context.Canceled) {
			slog.Error("[goutils.ai] AWS Bedrock Request canceled", "error", ctx.Err())
			return nil, fmt.Errorf("request canceled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("request failed: %w", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, context.Canceled) {
				slog.Error("[goutils.ai] AWS Bedrock Request canceled", "error", result.err)
				return nil, fmt.Errorf("request canceled: %w", result.err)
			}
			slog.Error("[goutils.ai] AWS Bedrock Request error", "error", result.err)
			return nil, result.err
		}
		return result.resp, nil
	}
}

func (s *Instant) CreateEmbeddingBedrock(ctx context.Context, input []string) ([]float32, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	// Prepare request payload for Cohere embed model
	payload := map[string]interface{}{
		"texts":           input,
		"input_type":      "search_document",
		"embedding_types": []string{"float"},
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		slog.Error("[goutils.ai] CreateEmbeddingBedrock marshal error", "error", err)
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	resp, err := s.bedrockClient.InvokeModelWithContext(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(s.cfg.AwsBedrockEmbeddingModelArn),
		Body:        bodyBytes,
		Accept:      aws.String("application/json"),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		slog.Error("[goutils.ai] CreateEmbeddingBedrock error", "error", err)
		return nil, err
	}

	var result struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		slog.Error("[goutils.ai] CreateEmbeddingBedrock unmarshal error", "error", err)
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(result.Embeddings) > 0 {
		return result.Embeddings[0], nil
	}

	return nil, nil
}
