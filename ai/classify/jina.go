package rerank

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
	JinaClassifyInputText struct {
		Model  string   `json:"model"`
		Input  []string `json:"input"`
		Labels []string `json:"labels"`
	}
	JinaClassifyInput struct {
		Model  string              `json:"model"`
		Input  []ClassifyInputItem `json:"input"`
		Labels []string            `json:"labels"`
	}
)

func (i2 *JinaClassifyInputText) Loads(i1 *ClassifyInput) {
	i2.Model = i1.Model
	for _, item := range i1.Input {
		i2.Input = append(i2.Input, item.Text)
	}
	i2.Labels = i1.Labels
}

func (i2 *JinaClassifyInput) Loads(i1 *ClassifyInput) {
	i2.Model = i1.Model
	i2.Input = i1.Input
	i2.Labels = i1.Labels
}

func JinaClassify(ctx context.Context, token, base string, input *ClassifyInput) (*ClassifyOutput, error) {
	var (
		textInput *JinaClassifyInputText
		data      []byte
		err       error
	)
	if input.Model == "jina-embeddings-v3" {
		textInput = &JinaClassifyInputText{}
		textInput.Loads(input)
		data, err = json.Marshal(textInput)
		if err != nil {
			return nil, err
		}
	} else {
		newInput := &JinaClassifyInput{}
		newInput.Loads(input)
		data, err = json.Marshal(newInput)
		if err != nil {
			return nil, err
		}
	}

	if base == "" {
		base = core.JINA_API_BASE
	}
	url := fmt.Sprintf("%s/v1/classify", base)
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

	output := &ClassifyOutput{}
	if err := json.Unmarshal(respData, &output); err != nil {
		return nil, err
	}

	return output, nil
}
