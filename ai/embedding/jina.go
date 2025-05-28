package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/lyricat/goutils/ai/core"
)

type (
	JinaCreateEmbeddingsInput struct {
		Model         string   `json:"model"`
		Task          string   `json:"task"`
		Input         []string `json:"input"`
		Truncate      bool     `json:"truncate,omitempty"`
		LateChunking  bool     `json:"late_chunking,omitempty"`
		Dimensions    int      `json:"dimensions,omitempty"`
		EmbeddingType string   `json:"embedding_type,omitempty"`
	}
)

func (i2 *JinaCreateEmbeddingsInput) Loads(i1 CreateEmbeddingsInput) {
	i2.Model = i1.Model
	i2.Input = i1.Input
	i2.Task = i1.JinaOptions.GetString("task")
	if i2.Task == "" {
		i2.Task = "text-matching"
	}
	i2.Truncate = i1.JinaOptions.GetBool("truncate")
	i2.LateChunking = i1.JinaOptions.GetBool("late_chunking")
	i2.Dimensions = int(i1.JinaOptions.GetInt64("dimensions"))
	if i2.Dimensions == 0 {
		i2.Dimensions = 1024
	}
	i2.EmbeddingType = "base64"
}

func JinaCreateEmbeddings(ctx context.Context, token, base string, input *JinaCreateEmbeddingsInput) (*CreateEmbeddingsOutput, error) {
	if base == "" {
		base = core.JINA_API_BASE
	}

	url := fmt.Sprintf("%s/embeddings", base)
	data, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

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
		return nil, fmt.Errorf("jina API request failed with status %d: %s", resp.StatusCode, string(respData))
	}

	output := &CreateEmbeddingsOutput{}
	if err := json.Unmarshal(respData, &output); err != nil {
		return nil, err
	}

	return output, nil
}
