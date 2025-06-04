package image

import (
	"context"
	"encoding/json"

	"github.com/lyricat/goutils/ai/util"
)

const (
	OpenAIAPIBase = "https://api.openai.com"
)

type (
	OpenAICreateImagesInput struct {
		Model             string `json:"model"`
		Prompt            string `json:"prompt"`
		Background        string `json:"background,omitempty"`
		Moderation        string `json:"moderation,omitempty"`
		N                 int    `json:"n"`
		OutputCompression string `json:"output_compression,omitempty"`
		OutputFormat      string `json:"output_format,omitempty"`
		Quality           string `json:"quality,omitempty"`
		Size              string `json:"size"`
	}

	OpenAICreateImagesUsage struct {
		InputTokens        int `json:"input_tokens"`
		OutputTokens       int `json:"output_tokens"`
		TotalTokens        int `json:"total_tokens"`
		InputTokensDetails struct {
			ImageTokens int `json:"image_tokens"`
			TextTokens  int `json:"text_tokens"`
		} `json:"input_tokens_details"`
	}

	OpenAICreateImagesOutput struct {
		Created int `json:"created"`
		Data    []struct {
			B64JSON string `json:"b64_json"`
		} `json:"data"`
		OpenAICreateImagesUsage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
			TotalTokens  int `json:"total_tokens"`
		} `json:"usage"`
	}

	OpenAIErrorResponse struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Param   string `json:"param"`
		} `json:"error"`
	}
)

func (i2 *OpenAICreateImagesInput) Loads(i1 *CreateImagesInput) {
	i2.Model = i1.Model
	i2.Prompt = i1.Prompt
	i2.N = i1.Count
	i2.Quality = i1.OpenAIOptions.GetString("quality")
	if i2.Quality == "" {
		i2.Quality = "low"
	}
	i2.Size = i1.OpenAIOptions.GetString("size")
	if i2.Size == "" {
		i2.Size = "1024x1024"
	}
	i2.Background = i1.OpenAIOptions.GetString("background")
	if i2.Background == "" {
		i2.Background = "auto"
	}
	i2.OutputFormat = i1.OpenAIOptions.GetString("output_format")
	if i2.OutputFormat == "" {
		i2.OutputFormat = "webp"
	}
}

func (m *OpenAICreateImagesOutput) ToCreateImagesOutput(input *OpenAICreateImagesInput) *CreateImagesOutput {
	return &CreateImagesOutput{
		Created: m.Created,
		Data:    m.Data,
		Usage: CreateImageUsage{
			Size:         input.Size,
			Quality:      input.Quality,
			InputTokens:  m.OpenAICreateImagesUsage.InputTokens,
			OutputTokens: m.OpenAICreateImagesUsage.OutputTokens,
			TotalTokens:  m.OpenAICreateImagesUsage.TotalTokens,
		},
		MimeType: getMimeType(input.OutputFormat),
	}
}

func OpenAICreateImages(ctx context.Context, token string, base string, input *CreateImagesInput) (*CreateImagesOutput, error) {
	openaiInput := &OpenAICreateImagesInput{}
	openaiInput.Loads(input)

	reqData, err := json.Marshal(openaiInput)
	if err != nil {
		return nil, err
	}

	respData, err := util.OpenAIRequest(ctx, token, base, "POST", "/images/generations", reqData)
	if err != nil {
		return nil, err
	}

	var resp *OpenAICreateImagesOutput
	err = json.Unmarshal(respData, &resp)
	if err != nil {
		return nil, err
	}

	return resp.ToCreateImagesOutput(openaiInput), nil
}

func getMimeType(format string) string {
	switch format {
	case "webp":
		return "image/webp"
	case "png":
		return "image/png"
	case "jpg":
		return "image/jpeg"
	case "jpeg":
		return "image/jpeg"
	default:
		return "image/webp"
	}
}
