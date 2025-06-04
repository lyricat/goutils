package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lyricat/goutils/ai/core"
)

type (
	// Anthropic Messages API Input & Output
	// FYI: https://docs.anthropic.com/en/api/messages
	AnthropicChatMessage struct {
		Role    string                    `json:"role"`
		Content []AnthropicMessageContent `json:"content,omitempty"`
	}

	AnthropicCacheControl struct {
		Type string `json:"type"` // "persistent" or "ephemeral"
	}

	AnthropicMessageContent struct {
		Type         string                 `json:"type"`
		Text         string                 `json:"text,omitempty"`
		CacheControl *AnthropicCacheControl `json:"cache_control,omitempty"`
	}

	AnthropicRequest struct {
		Model         string                 `json:"model"`
		Messages      []AnthropicChatMessage `json:"messages"`
		MaxTokens     int                    `json:"max_tokens"`
		System        string                 `json:"system,omitempty"`
		Temperature   float64                `json:"temperature,omitempty"`
		TopP          float64                `json:"top_p,omitempty"`
		TopK          int                    `json:"top_k,omitempty"`
		StopSequences []string               `json:"stop_sequences,omitempty"`
	}

	AnthropicResponse struct {
		ID           string                    `json:"id"`
		Type         string                    `json:"type"`
		Role         string                    `json:"role"`
		Content      []AnthropicMessageContent `json:"content"`
		Model        string                    `json:"model"`
		StopReason   string                    `json:"stop_reason"`
		StopSequence *string                   `json:"stop_sequence"`
		Usage        struct {
			InputTokens              int `json:"input_tokens"`
			OutputTokens             int `json:"output_tokens"`
			CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		} `json:"usage"`
	}
)

func (s *Instant) AnthropicRawRequest(ctx context.Context, messages []AnthropicChatMessage, _opts *core.RawRequestOptions) (*core.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*600)
	defer cancel()

	resultChan := make(chan struct {
		resp *core.Result
		err  error
	})

	go func() {
		body := AnthropicRequest{
			Model:     s.cfg.AnthropicModel,
			Messages:  messages,
			MaxTokens: 8192,
		}

		bodyBytes, err := json.Marshal(body)
		if err != nil {
			resultChan <- struct {
				resp *core.Result
				err  error
			}{resp: nil, err: fmt.Errorf("failed to marshal request body: %w", err)}
			return
		}

		req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", strings.NewReader(string(bodyBytes)))
		if err != nil {
			resultChan <- struct {
				resp *core.Result
				err  error
			}{resp: nil, err: fmt.Errorf("failed to create request: %w", err)}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", s.cfg.AnthropicAPIKey)
		req.Header.Set("anthropic-version", "2023-06-01")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			resultChan <- struct {
				resp *core.Result
				err  error
			}{resp: nil, err: fmt.Errorf("failed to send request: %w", err)}
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			var errorResponse map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
				resultChan <- struct {
					resp *core.Result
					err  error
				}{resp: nil, err: fmt.Errorf("API error: status code %d", resp.StatusCode)}
				return
			}
			resultChan <- struct {
				resp *core.Result
				err  error
			}{resp: nil, err: fmt.Errorf("API error: %v", errorResponse)}
			return
		}

		var r AnthropicResponse
		if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
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
			}{resp: nil, err: errors.New("empty response content")}
			return
		}

		result := &core.Result{
			Text: r.Content[0].Text,
		}

		// Handle usage information including cache tokens
		result.Usage.InputTokens = r.Usage.InputTokens
		result.Usage.OutputTokens = r.Usage.OutputTokens
		// Add cache tokens if available
		result.Usage.CacheCreationInputTokens = r.Usage.CacheCreationInputTokens
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
			slog.Error("[goutils.ai] Anthropic Request canceled", "error", ctx.Err())
			return nil, fmt.Errorf("request canceled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("request failed: %w", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, context.Canceled) {
				slog.Error("[goutils.ai] Anthropic Request canceled", "error", result.err)
				return nil, fmt.Errorf("request canceled: %w", result.err)
			}
			slog.Error("[goutils.ai] Anthropic Request error", "error", result.err)
			return nil, result.err
		}
		return result.resp, nil
	}
}

// Helper function to create a chat message with cache control
func CreateAnthropicMessageWithCache(role, text string, cacheType string) AnthropicChatMessage {
	var content []AnthropicMessageContent

	if cacheType != "" {
		content = []AnthropicMessageContent{
			{
				Type: "text",
				Text: text,
				CacheControl: &AnthropicCacheControl{
					Type: cacheType, // "persistent" or "ephemeral"
				},
			},
		}
	} else {
		content = []AnthropicMessageContent{
			{
				Type: "text",
				Text: text,
			},
		}
	}

	return AnthropicChatMessage{
		Role:    role,
		Content: content,
	}
}
