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
	RerankInputText struct {
		Model           string   `json:"model"`
		Query           string   `json:"query"`
		Documents       []string `json:"documents"`
		TopN            int      `json:"top_n"`
		ReturnDocuments bool     `json:"return_documents"`
	}
)

func (i2 *RerankInputText) Loads(i1 *RerankInput) {
	i2.Model = i1.Model
	for _, item := range i1.Documents {
		i2.Documents = append(i2.Documents, item.Text)
	}
	i2.TopN = i1.TopN
	i2.ReturnDocuments = i1.ReturnDocuments
	i2.Query = i1.Query
}

func JinaRerank(ctx context.Context, token, base string, input *RerankInput) (*RerankOutput, error) {
	var (
		textInput *RerankInputText
		data      []byte
		err       error
	)
	if input.Model == "jina-reranker-v2-base-multilingual" {
		textInput = &RerankInputText{}
		textInput.Loads(input)
		data, err = json.Marshal(textInput)
		if err != nil {
			return nil, err
		}
	} else {
		data, err = json.Marshal(input)
		if err != nil {
			return nil, err
		}
	}

	if base == "" {
		base = core.JINA_API_BASE
	}
	url := fmt.Sprintf("%s/v1/rerank", base)
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

	output := &RerankOutput{}
	if err := json.Unmarshal(respData, &output); err != nil {
		return nil, err
	}

	return output, nil
}
