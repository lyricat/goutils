package gemini

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/http"
)

type (
	ImageGenerationModel string

	AspectRatio string

	SafetyFilterLevel string

	PersonGeneration string

	ResponseModality string

	ImageGenerationOptions struct {
		// Model to use for image generation
		Model ImageGenerationModel

		// For both models:
		Prompt string

		// For Gemini model:
		ResponseModalities []ResponseModality

		// For Imagen model:
		NumberOfImages    int
		AspectRatio       AspectRatio
		SafetyFilterLevel SafetyFilterLevel
		PersonGeneration  PersonGeneration
	}

	ImageResponse struct {
		Images []image.Image
		Text   string
	}

	GeminiInstant struct {
		APIKey string
	}
)

// ImageGenerationModel specifies which model to use for image generation

const (
	// GeminiFlashExp is the Gemini 2.0 Flash Experimental model for native image generation
	GeminiFlashExp ImageGenerationModel = "gemini-2.0-flash-exp-image-generation"
	// Imagen3 is Google's highest quality text-to-image model
	Imagen3 ImageGenerationModel = "imagen-3.0-generate-002"
)

const (
	AspectRatioSquare       AspectRatio = "1:1"
	AspectRatioPortrait34   AspectRatio = "3:4"
	AspectRatioLandscape43  AspectRatio = "4:3"
	AspectRatioPortrait916  AspectRatio = "9:16"
	AspectRatioLandscape169 AspectRatio = "16:9"
)

const (
	BlockLowAndAbove    SafetyFilterLevel = "BLOCK_LOW_AND_ABOVE"
	BlockMediumAndAbove SafetyFilterLevel = "BLOCK_MEDIUM_AND_ABOVE"
	BlockOnlyHigh       SafetyFilterLevel = "BLOCK_ONLY_HIGH"
)

const (
	DontAllow  PersonGeneration = "DONT_ALLOW"
	AllowAdult PersonGeneration = "ALLOW_ADULT"
)

const (
	TextModality  ResponseModality = "Text"
	ImageModality ResponseModality = "Image"
)

func New(apiKey string) *GeminiInstant {
	return &GeminiInstant{
		APIKey: apiKey,
	}
}

// GeminiGenerateImage generates images using the Gemini API
func (c *GeminiInstant) GeminiGenerateImage(ctx context.Context, opts ImageGenerationOptions) (*ImageResponse, error) {
	if opts.Model == "" {
		opts.Model = GeminiFlashExp
	}

	if opts.Model == GeminiFlashExp {
		return c.generateGeminiImage(ctx, opts)
	} else if opts.Model == Imagen3 {
		return c.generateImagenImage(ctx, opts)
	}

	return nil, fmt.Errorf("unsupported model: %s", opts.Model)
}

// generateGeminiImage generates images using the Gemini 2.0 Flash Experimental model
func (c *GeminiInstant) generateGeminiImage(ctx context.Context, opts ImageGenerationOptions) (*ImageResponse, error) {
	if len(opts.ResponseModalities) == 0 {
		opts.ResponseModalities = []ResponseModality{TextModality, ImageModality}
	}

	// Prepare request body
	reqBody := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{
						"text": opts.Prompt,
					},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"responseModalities": opts.ResponseModalities,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		opts.Model, c.APIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text       string `json:"text,omitempty"`
					InlineData struct {
						MimeType string `json:"mimeType"`
						Data     string `json:"data"`
					} `json:"inlineData,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount    int `json:"promptTokenCount"`
			TotalTokenCount     int `json:"TotalTokenCount"`
			PromptTokensDetails []struct {
				Modality   string `json:"modality"`
				TokenCount int    `json:"tokenCount"`
			} `json:"promptTokensDetails"`
		} `json:"usageMetadata"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := &ImageResponse{
		Images: []image.Image{},
	}

	if len(response.Candidates) > 0 {
		for _, part := range response.Candidates[0].Content.Parts {
			if part.Text != "" {
				result.Text += part.Text
			} else if part.InlineData.Data != "" {
				imgData, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					return nil, fmt.Errorf("failed to decode image data: %w", err)
				}

				img, err := png.Decode(bytes.NewBuffer(imgData))
				if err != nil {
					return nil, fmt.Errorf("failed to decode image: %w", err)
				}

				result.Images = append(result.Images, img)
			}
		}
	}

	return result, nil
}

func (c *GeminiInstant) generateImagenImage(ctx context.Context, opts ImageGenerationOptions) (*ImageResponse, error) {
	// Set defaults
	if opts.NumberOfImages <= 0 || opts.NumberOfImages > 4 {
		opts.NumberOfImages = 4
	}

	// Prepare request body
	reqBody := map[string]interface{}{
		"instances": []map[string]interface{}{
			{
				"prompt": opts.Prompt,
			},
		},
		"parameters": map[string]interface{}{
			"sampleCount": opts.NumberOfImages,
		},
	}

	if opts.AspectRatio != "" {
		reqBody["parameters"].(map[string]interface{})["aspectRatio"] = opts.AspectRatio
	}

	if opts.SafetyFilterLevel != "" {
		reqBody["parameters"].(map[string]interface{})["safetyFilterLevel"] = opts.SafetyFilterLevel
	}

	if opts.PersonGeneration != "" {
		reqBody["parameters"].(map[string]interface{})["personGeneration"] = opts.PersonGeneration
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:predict?key=%s",
		opts.Model, c.APIKey)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Predictions []struct {
			BytesBase64Encoded string `json:"bytesBase64Encoded"`
			MimeType           string `json:"mimeType"`
		} `json:"predictions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	result := &ImageResponse{
		Images: []image.Image{},
	}

	if len(response.Predictions) > 0 {
		for _, item := range response.Predictions {
			imgData, err := base64.StdEncoding.DecodeString(item.BytesBase64Encoded)
			if err != nil {
				return nil, fmt.Errorf("failed to decode image data: %w", err)
			}

			imgObj, err := png.Decode(bytes.NewBuffer(imgData))
			if err != nil {
				return nil, fmt.Errorf("failed to decode image: %w", err)
			}

			result.Images = append(result.Images, imgObj)
		}
	}

	return result, nil
}
