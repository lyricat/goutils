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
		Model         string `json:"model"`
		EmbeddingType string `json:"embedding_type,omitempty"`
		Dimensions    int    `json:"dimensions,omitempty"`
		Task          string `json:"task,omitempty"`

		Input        []string `json:"input"`
		Truncate     bool     `json:"truncate,omitempty"`
		LateChunking bool     `json:"late_chunking,omitempty"`
	}

	JinaCreateEmbeddingsClipInput struct {
		Model         string `json:"model"`
		EmbeddingType string `json:"embedding_type,omitempty"`
		Dimensions    int    `json:"dimensions,omitempty"`
		Task          string `json:"task,omitempty"`

		Input      []CreateEmbeddingsInputItem `json:"input"`
		Normalized bool                        `json:"normalized,omitempty"`
	}
)

func (i2 *JinaCreateEmbeddingsInput) Loads(i1 *CreateEmbeddingsInput) {
	i2.Model = i1.Model
	for _, item := range i1.Input {
		i2.Input = append(i2.Input, item.Text)
	}
	i2.Task = i1.JinaOptions.GetString("task")
	if i2.Task == "" {
		i2.Task = "text-matching"
	}
	if i2.Dimensions == 0 {
		i2.Dimensions = 1024
	}
	i2.EmbeddingType = "base64"

	i2.Truncate = i1.JinaOptions.GetBool("truncate")
	i2.LateChunking = i1.JinaOptions.GetBool("late_chunking")
	i2.Dimensions = int(i1.JinaOptions.GetInt64("dimensions"))
}

func (i2 *JinaCreateEmbeddingsClipInput) Loads(i1 *CreateEmbeddingsInput) {
	i2.Model = i1.Model
	i2.Input = i1.Input
	i2.Task = i1.JinaOptions.GetString("task")
	if i2.Task == "" {
		i2.Task = "text-matching"
	}
	i2.Dimensions = int(i1.JinaOptions.GetInt64("dimensions"))
	i2.EmbeddingType = "base64"

	if i2.Dimensions == 0 {
		i2.Dimensions = 1024
	}
	i2.Normalized = i1.JinaOptions.GetBool("normalized")
}

func JinaCreateEmbeddings(ctx context.Context, token, base string, input *CreateEmbeddingsInput) (*CreateEmbeddingsOutput, error) {
	var (
		jinaInput     *JinaCreateEmbeddingsInput
		jinaClipInput *JinaCreateEmbeddingsClipInput
		data          []byte
		err           error
	)
	if input.Model == "jina-clip-v2" {
		jinaClipInput = &JinaCreateEmbeddingsClipInput{}
		jinaClipInput.Loads(input)
		data, err = json.Marshal(jinaClipInput)
		if err != nil {
			return nil, err
		}
	} else {
		jinaInput = &JinaCreateEmbeddingsInput{}
		jinaInput.Loads(input)
		data, err = json.Marshal(jinaInput)
		if err != nil {
			return nil, err
		}
	}

	if base == "" {
		base = core.JINA_API_BASE
	}
	url := fmt.Sprintf("%s/embeddings", base)
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
