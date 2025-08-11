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
	GeminiAPIBase = "https://generativelanguage.googleapis.com"
)

const (
	GeminiEmbeddingTaskTypeSemanticSimilarity = "SEMANTIC_SIMILARITY"
	GeminiEmbeddingTaskTypeClassification     = "CLASSIFICATION"
	GeminiEmbeddingTaskTypeClustering         = "CLUSTERING"
	GeminiEmbeddingTaskTypeRetrievalDocument  = "RETRIEVAL_DOCUMENT"
	GeminiEmbeddingTaskTypeRetrievalQuery     = "RETRIEVAL_QUERY"
	GeminiEmbeddingTaskTypeCodeRetrievalQuery = "CODE_RETRIEVAL_QUERY"
	GeminiEmbeddingTaskTypeQuestionAnswering  = "QUESTION_ANSWERING"
	GeminiEmbeddingTaskTypeFactVerification   = "FACT_VERIFICATION"
)

type (
	GeminiCreateEmbeddingsInput struct {
		Model   string        `json:"model"`
		Content GeminiContent `json:"content"`
		GeminiEmbeddingConfig
	}

	GeminiContent struct {
		Parts []GeminiPart `json:"parts"`
	}

	GeminiPart struct {
		Text string `json:"text"`
	}

	GeminiEmbeddingConfig struct {
		TaskType             string `json:"task_type,omitempty"`
		OutputDimensionality int    `json:"output_dimensionality,omitempty"`
	}

	GeminiCreateEmbeddingsOutput struct {
		Embedding GeminiEmbedding `json:"embedding"`
	}

	GeminiEmbedding struct {
		Values []float64 `json:"values"`
	}
)

func (i2 *GeminiCreateEmbeddingsInput) Loads(i1 *CreateEmbeddingsInput) {
	for _, item := range i1.Input {
		content := GeminiPart{
			Text: item.Text,
		}
		i2.Content.Parts = append(i2.Content.Parts, content)
	}

	// Set up embedding config from options
	config := &GeminiEmbeddingConfig{}

	taskType := i1.GeminiOptions.GetString("task_type")
	if taskType != "" {
		config.TaskType = taskType
	}

	dimensions := int(i1.GeminiOptions.GetInt64("output_dimensionality"))
	if dimensions > 0 {
		config.OutputDimensionality = dimensions
	}

	if config.TaskType != "" || config.OutputDimensionality != 0 {
		i2.TaskType = config.TaskType
		i2.OutputDimensionality = config.OutputDimensionality
	}

	i2.Model = "models/gemini-embedding-001"

}

func GeminiCreateEmbeddings(ctx context.Context, token, base string, input *CreateEmbeddingsInput) (*CreateEmbeddingsOutput, error) {
	geminiInput := &GeminiCreateEmbeddingsInput{}
	geminiInput.Loads(input)

	data, err := json.Marshal(geminiInput)
	if err != nil {
		return nil, err
	}

	if base == "" {
		base = GeminiAPIBase
	}
	url := fmt.Sprintf("%s/v1beta/models/gemini-embedding-001:embedContent", base)

	fmt.Printf("url: %s\n", url)
	fmt.Printf("data: %s\n", string(data))

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", token)

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
		return nil, fmt.Errorf("gemini API request failed with status %d: %s", resp.StatusCode, string(respData))
	}

	geminiOutput := &GeminiCreateEmbeddingsOutput{}
	if err := json.Unmarshal(respData, geminiOutput); err != nil {
		return nil, err
	}

	// Convert to standard output format
	output := &CreateEmbeddingsOutput{
		Model:  "models/gemini-embedding-001",
		Object: "list",
		Data: make([]struct {
			Object    string `json:"object"`
			Embedding string `json:"embedding"`
			Index     int    `json:"index"`
		}, len(geminiOutput.Embedding.Values)),
	}

	// for i, vs := range geminiOutput.Embedding.Values {
	// 	// Convert float64 slice to base64-encoded JSON string to match the expected format
	// 	if len(vs) == 0 {
	// 		continue
	// 	}

	// 	// 4 bytes per float32
	// 	buf := make([]byte, len(emb)*4)
	// 	for i, v := range emb {
	// 		offset := i * 4
	// 		binary.LittleEndian.PutUint32(buf[offset:], math.Float32bits(v))
	// 	}
	// 	based64Data, err := base64.StdEncoding.EncodeToString(vs)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("failed to encode embedding data: %w", err)
	// 	}
	// 	output.Data[i] = struct {
	// 		Object    string `json:"object"`
	// 		Embedding string `json:"embedding"`
	// 		Index     int    `json:"index"`
	// 	}{
	// 		Object:    "embedding",
	// 		Embedding: based64Data,
	// 		Index:     i,
	// 	}
	// }

	return output, nil
}
