package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

type (
	DeepseekRawRequestOptions struct {
		UseJSON bool
	}

	DeepseekChatPayloadResponseFormat struct {
		Type string `json:"type"`
	}

	DeepseekChatPayload struct {
		Messages         []GeneralChatCompletionMessage    `json:"messages"`
		Model            string                            `json:"model"`
		ResponseFormat   DeepseekChatPayloadResponseFormat `json:"response_format"`
		Temperature      float64                           `json:"temperature"`
		MaxTokens        int                               `json:"max_tokens"`
		FrequencyPenalty float64                           `json:"frequency_penalty"`
		PresencePenalty  float64                           `json:"presence_penalty"`
		Stop             interface{}                       `json:"stop"`
		Stream           bool                              `json:"stream"`
		StreamOptions    interface{}                       `json:"stream_options"`
		TopP             float64                           `json:"top_p"`
		Tools            interface{}                       `json:"tools"`
		ToolChoice       string                            `json:"tool_choice"`
		Logprobs         bool                              `json:"logprobs"`
		TopLogprobs      interface{}                       `json:"top_logprobs"`
	}

	DeepseekChatResponseChoice struct {
		Index        int                          `json:"index"`
		Message      GeneralChatCompletionMessage `json:"message"`
		Logprobs     interface{}                  `json:"logprobs"`
		FinishReason string                       `json:"finish_reason"`
	}

	DeepseekResponseUsage struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		TotalTokens         int `json:"total_tokens"`
		PromptTokensDetails struct {
			CachedTokens int `json:"cached_tokens"`
		} `json:"prompt_tokens_details"`
		PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens"`
		PromptCacheMissTokens int `json:"prompt_cache_miss_tokens"`
	}

	DeepseekErrorResponse struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Param   string `json:"param"`
			Code    string `json:"code"`
		} `json:"error"`
	}

	DeepseekChatResponse struct {
		DeepseekErrorResponse
		ID                string                       `json:"id"`
		Object            string                       `json:"object"`
		Created           int64                        `json:"created"`
		Model             string                       `json:"model"`
		Choices           []DeepseekChatResponseChoice `json:"choices"`
		Usage             DeepseekResponseUsage        `json:"usage"`
		SystemFingerprint string                       `json:"system_fingerprint"`
	}
)

func (s *Instant) DeepseekRawRequest(ctx context.Context, messages []GeneralChatCompletionMessage, opts *DeepseekRawRequestOptions) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*180)
	defer cancel()

	resultChan := make(chan struct {
		resp string
		err  error
	})

	go func() {
		payload := DeepseekChatPayload{
			Messages:    messages,
			Model:       s.cfg.DeepseekModel,
			MaxTokens:   4096,
			Stop:        nil,
			Tools:       nil,
			TopP:        1,
			Temperature: 1,
			ToolChoice:  "none",
			Logprobs:    false,
		}

		if opts != nil {
			if opts.UseJSON && supportJSONResponse(s.cfg.DeepseekModel) {
				payload.ResponseFormat.Type = "json_object"
			} else {
				payload.ResponseFormat.Type = "text"
			}
		}

		payloadJson, err := json.Marshal(payload)
		if err != nil {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: fmt.Errorf("failed to marshal payload: %w", err)}
			return
		}

		buf := bytes.NewBuffer(payloadJson)
		apiUrl := fmt.Sprintf("%s/chat/completions", s.cfg.DeepseekEndpoint)

		req, err := http.NewRequest(http.MethodPost, apiUrl, buf)
		if err != nil {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: fmt.Errorf("failed to create request: %w", err)}
			return
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", s.cfg.DeepseekApiKey))

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: fmt.Errorf("failed to send request: %w", err)}
			return
		}
		defer resp.Body.Close()
		// buf1, _ := io.ReadAll(resp.Body)
		// fmt.Printf("buf: %v\n", string(buf1))

		var body DeepseekChatResponse
		if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: fmt.Errorf("failed to decode response: %w", err)}
			return
		}

		if body.Error.Code != "" {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: fmt.Errorf("deepseek error: %s, %s", body.Error.Code, body.Error.Message)}
			return
		}

		if len(body.Choices) == 0 {
			resultChan <- struct {
				resp string
				err  error
			}{resp: "", err: fmt.Errorf("no choices in response")}
			return
		}

		resultChan <- struct {
			resp string
			err  error
		}{resp: body.Choices[0].Message.Content, err: nil}
	}()

	select {
	case <-ctx.Done():
		// Context was canceled or timed out
		if errors.Is(ctx.Err(), context.Canceled) {
			slog.Error("[goutils.ai] deepseek request canceled", "error", ctx.Err())
			return "", fmt.Errorf("request canceled: %w", ctx.Err())
		}
		return "", fmt.Errorf("request failed: %w", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, context.Canceled) {
				slog.Error("[goutils.ai] deepseek request canceled", "error", result.err)
				return "", fmt.Errorf("request canceled: %w", result.err)
			}
			slog.Error("[goutils.ai] deepseek request error", "error", result.err)
			return "", result.err
		}
		return result.resp, nil
	}
}
