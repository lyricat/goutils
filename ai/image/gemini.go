package image

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"time"
)

type (
	GeminiCreateImagesInput struct {
		Model  string
		Prompt string

		NumberOfImages    int
		AspectRatio       string
		SafetyFilterLevel string
		PersonGeneration  string
	}

	GeminiCreateImagesOutput struct {
		Images []image.Image
		Text   string
	}
)

const (
	AspectRatioSquare       = "1:1"
	AspectRatioPortrait34   = "3:4"
	AspectRatioLandscape43  = "4:3"
	AspectRatioPortrait916  = "9:16"
	AspectRatioLandscape169 = "16:9"
)

const (
	BlockLowAndAbove    = "BLOCK_LOW_AND_ABOVE"
	BlockMediumAndAbove = "BLOCK_MEDIUM_AND_ABOVE"
	BlockOnlyHigh       = "BLOCK_ONLY_HIGH"
)

const (
	DontAllow = "DONT_ALLOW"
	Allow     = "ALLOW_ADULT"
)

const (
	TextModality  = "Text"
	ImageModality = "Image"
)

func (i2 *GeminiCreateImagesInput) Loads(i1 *CreateImagesInput) {
	i2.Model = i1.Model
	i2.Prompt = i1.Prompt
	i2.NumberOfImages = i1.Count
	if i2.NumberOfImages <= 0 || i2.NumberOfImages > 4 {
		i2.NumberOfImages = 1
	}
	i2.AspectRatio = i1.GeminiOptions.GetString("aspect_ratio")
	if i2.AspectRatio == "" {
		i2.AspectRatio = AspectRatioSquare
	}
	i2.SafetyFilterLevel = i1.GeminiOptions.GetString("safety_filter_level")
	if i2.SafetyFilterLevel == "" {
		i2.SafetyFilterLevel = BlockOnlyHigh
	}
	i2.PersonGeneration = i1.GeminiOptions.GetString("person_generation")
	if i2.PersonGeneration == "" {
		i2.PersonGeneration = Allow
	}
}

func GeminiCreateImages(ctx context.Context, token string, input *CreateImagesInput) (*CreateImagesOutput, error) {
	if input.Model == "" {
		input.Model = "imagen-3.0-generate-002"
	}
	if input.Model != "imagen-3.0-generate-002" {
		return nil, fmt.Errorf("invalid model: %s", input.Model)
	}

	geminiInput := &GeminiCreateImagesInput{}
	geminiInput.Loads(input)

	// Prepare request body
	reqBody := map[string]any{
		"instances": []map[string]any{
			{
				"prompt": geminiInput.Prompt,
			},
		},
		"parameters": map[string]any{
			"sampleCount":       geminiInput.NumberOfImages,
			"aspectRatio":       geminiInput.AspectRatio,
			"safetyFilterLevel": geminiInput.SafetyFilterLevel,
			"personGeneration":  geminiInput.PersonGeneration,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:predict?key=%s", geminiInput.Model, token)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var data struct {
		Predictions []struct {
			BytesBase64Encoded string `json:"bytesBase64Encoded"`
			MimeType           string `json:"mimeType"`
		} `json:"predictions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	now := time.Now()
	result := &CreateImagesOutput{
		Created: int(now.Unix()),
	}

	for _, item := range data.Predictions {
		result.Data = append(result.Data, struct {
			B64JSON string `json:"b64_json"`
		}{
			B64JSON: item.BytesBase64Encoded,
		})
	}

	if len(data.Predictions) > 0 {
		result.MimeType = data.Predictions[0].MimeType
	}

	return result, nil
}
