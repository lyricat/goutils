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

	"github.com/lyricat/goutils/ai/core"
)

type (
	SusanoParams struct {
		Format     string                 `json:"format"`
		Search     SusanoParamsSearch     `json:"search"`
		Conditions SusanoParamsConditions `json:"conditions"`
	}

	SusanoParamsSearch struct {
		Enabled bool `json:"enabled"`
		Limit   int  `json:"limit"`
	}

	SusanoParamsConditions struct {
		PreferredProvider string `json:"preferred_provider"`
		PreferredModel    string `json:"preferred_model"`
	}

	SusanooTaskRequest struct {
		Messages []core.Message `json:"messages"`
		Params   map[string]any `json:"params"`
	}

	SusanooTaskResponse struct {
		Data struct {
			Code    int    `json:"code"`
			TraceID string `json:"trace_id"`
		} `json:"data"`
		Ts int `json:"ts"`
	}

	SusanooTaskResultResponse struct {
		Data struct {
			ID           int            `json:"id"`
			ProxyID      int            `json:"proxy_id"`
			Result       map[string]any `json:"result"`
			Status       int            `json:"status"`
			TraceID      string         `json:"trace_id"`
			ScheduledAt  string         `json:"scheduled_at"`
			CreatedAt    string         `json:"created_at"`
			UpdatedAt    string         `json:"updated_at"`
			PendingCount int            `json:"pending_count"`
		} `json:"data"`
		Ts int `json:"ts"`
	}
)

func (s *Instant) SusanooRawRequest(ctx context.Context, messages []core.Message, params map[string]any) (*SusanooTaskResultResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*5)
	defer cancel()

	resultChan := make(chan struct {
		result *SusanooTaskResultResponse
		err    error
	})

	go func() error {
		task := &SusanooTaskRequest{
			Messages: messages,
			Params:   params,
		}
		if task.Params == nil {
			task.Params = make(map[string]any)
		}

		traceID, err := s.SusanooCreateTask(ctx, task)
		if err != nil {
			return err
		}

		for {
			result, err := s.SusanooFetchTaskResult(ctx, traceID)
			if err != nil {
				return err
			}
			if result.Data.Status == 1 || result.Data.Status == 2 {
				// 1, assigned, but not started
				// 2, in progress
				time.Sleep(time.Second * 3)
				continue
			}
			if result.Data.Status == 3 || result.Data.Status == 4 {
				// 3, finished
				// 4, failed
				resultChan <- struct {
					result *SusanooTaskResultResponse
					err    error
				}{result: result, err: nil}
				return nil
			}
		}
	}()

	select {
	case <-ctx.Done():
		// Context was canceled or timed out
		if errors.Is(ctx.Err(), context.Canceled) {
			slog.Error("[goutils.ai] Susanoo Request canceled", "error", ctx.Err())
			return nil, fmt.Errorf("request canceled: %w", ctx.Err())
		}
		return nil, fmt.Errorf("request failed: %w", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, context.Canceled) {
				slog.Error("[goutils.ai] Susanoo Request canceled", "error", result.err)
				return nil, fmt.Errorf("request canceled: %w", result.err)
			}
			slog.Error("[goutils.ai] Susanoo Request error", "error", result.err)
			return nil, result.err
		}
		return result.result, nil
	}
}

func (s *Instant) SusanooCreateTask(ctx context.Context, task *SusanooTaskRequest) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*60)
	defer cancel()

	resultChan := make(chan struct {
		traceID string
		err     error
	})

	go func() error {
		payload, err := json.Marshal(task)
		if err != nil {
			return err
		}

		buf := bytes.NewBuffer(payload)
		apiUrl := fmt.Sprintf("%s/tasks", s.cfg.SusanooAPIBase)
		req, err := http.NewRequest(http.MethodPost, apiUrl, buf)
		if err != nil {
			return err
		}
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("X-SUSANOO-KEY", s.cfg.SusanooAPIKey)

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		var body SusanooTaskResponse
		if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
			return err
		}

		if body.Data.Code != 0 {
			return fmt.Errorf("failed to create task at susanoo: %d", body.Data.Code)
		}

		if err != nil {
			resultChan <- struct {
				traceID string
				err     error
			}{traceID: "", err: err}
			return err
		}

		resultChan <- struct {
			traceID string
			err     error
		}{traceID: body.Data.TraceID, err: nil}

		return nil
	}()

	select {
	case <-ctx.Done():
		// Context was canceled or timed out
		if errors.Is(ctx.Err(), context.Canceled) {
			slog.Error("[goutils.ai] Susanoo create task canceled", "error", ctx.Err())
			return "", fmt.Errorf("susanoo create task canceled: %w", ctx.Err())
		}
		return "", fmt.Errorf("request failed: %w", ctx.Err())
	case result := <-resultChan:
		if result.err != nil {
			if errors.Is(result.err, context.Canceled) {
				slog.Error("[goutils.ai] Susanoo create task canceled", "error", result.err)
				return "", fmt.Errorf("susanoo create task canceled: %w", result.err)
			}
			slog.Error("[goutils.ai] Susanoo Request error", "error", result.err)
			return "", result.err
		}
		return result.traceID, nil
	}
}

func (s *Instant) SusanooFetchTaskResult(ctx context.Context, traceID string) (*SusanooTaskResultResponse, error) {
	apiUrl := fmt.Sprintf("%s/tasks/result?trace_id=%s", s.cfg.SusanooAPIBase, traceID)
	req, err := http.NewRequest(http.MethodGet, apiUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-SUSANOO-KEY", s.cfg.SusanooAPIKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var body SusanooTaskResultResponse
	if err = json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}

	if body.Data.ID == 0 {
		return nil, fmt.Errorf("failed to fetch task result at susanoo")
	}

	return &body, nil
}

func (p *SusanoParams) ToMap() map[string]any {
	params := make(map[string]any)
	params["format"] = p.Format
	params["conditions"] = make(map[string]any)
	params["conditions"].(map[string]any)["preferred_provider"] = p.Conditions.PreferredProvider
	params["conditions"].(map[string]any)["preferred_model"] = p.Conditions.PreferredModel
	return params
}
