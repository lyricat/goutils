package util

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	OpenAIAPIBase = "https://api.openai.com"
)

func OpenAIRequest(ctx context.Context, token, base, method, url string, data []byte) ([]byte, error) {
	if base == "" {
		base = OpenAIAPIBase
	}
	apiUrl := fmt.Sprintf("%s%s", base, url)
	req, err := http.NewRequestWithContext(ctx, method, apiUrl, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return respData, fmt.Errorf("openai API request failed with status %d: %s", resp.StatusCode, string(respData))
	}

	return respData, nil
}
