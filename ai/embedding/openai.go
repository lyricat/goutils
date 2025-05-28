package embedding

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const (
	OpenAIAPIBase = "https://api.openai.com"
)

type (
	OpenAICreateEmbeddingsInput struct {
		Model          string   `json:"model"`
		Input          []string `json:"input"`
		EncodingFormat string   `json:"encoding_format,omitempty"`
		Dimensions     *int     `json:"dimensions,omitempty"`
		User           string   `json:"user,omitempty"`
	}
)

func (i2 *OpenAICreateEmbeddingsInput) Loads(i1 *CreateEmbeddingsInput) {
	i2.Model = i1.Model
	for _, item := range i1.Input {
		i2.Input = append(i2.Input, item.Text)
	}
	dim := int(i1.OpenAIOptions.GetInt64("dimensions"))
	i2.Dimensions = &dim
	if *i2.Dimensions == 0 {
		i2.Dimensions = nil
	}
	i2.EncodingFormat = "base64"
}

func OpenAICreateEmbeddings(ctx context.Context, token string, base string, input *CreateEmbeddingsInput) (*CreateEmbeddingsOutput, error) {
	openaiInput := &OpenAICreateEmbeddingsInput{}
	openaiInput.Loads(input)

	if base == "" {
		base = OpenAIAPIBase
	}
	url := fmt.Sprintf("%s/v1/embeddings", base)
	data, err := json.Marshal(openaiInput)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
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
		return nil, fmt.Errorf("openai API request failed with status %d: %s", resp.StatusCode, string(respData))
	}

	output := &CreateEmbeddingsOutput{}
	if err := json.Unmarshal(respData, &output); err != nil {
		return nil, err
	}

	if len(output.Data) == 0 {
		return nil, fmt.Errorf("no embeddings found in OpenAI API response")
	}

	return output, nil
}
