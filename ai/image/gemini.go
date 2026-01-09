package image

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net/http"
	"slices"
	"strings"
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

		// Native Gemini image generation (Nano Banana)
		ResponseModalities []string
		ImageSize          string
	}

	GeminiCreateImagesOutput struct {
		Images []image.Image
		Text   string
	}
)

const (
	AspectRatioSquare       = "1:1"
	AspectRatioPortrait23   = "2:3"
	AspectRatioLandscape32  = "3:2"
	AspectRatioPortrait34   = "3:4"
	AspectRatioLandscape43  = "4:3"
	AspectRatioPortrait45   = "4:5"
	AspectRatioLandscape54  = "5:4"
	AspectRatioPortrait916  = "9:16"
	AspectRatioLandscape169 = "16:9"
	AspectRatioLandscape219 = "21:9"
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
	ResponseModalityText        = "TEXT"
	ResponseModalityImage       = "IMAGE"
	ResponseModalityTextLegacy  = "Text"
	ResponseModalityImageLegacy = "Image"
)

const (
	// Imagen (predict endpoint)
	GeminiModelImagen3 = "imagen-3.0-generate-002"

	// Nano Banana native image generation (generateContent endpoint)
	GeminiModelNanoBanana    = "gemini-2.5-flash-image"
	GeminiModelNanoBananaPro = "gemini-3-pro-image-preview"
)

var (
	errInvalidGeminiModel = errors.New("invalid gemini image model")
)

func (i2 *GeminiCreateImagesInput) Loads(i1 *CreateImagesInput) {
	i2.Model = i1.Model
	i2.Prompt = i1.Prompt
	i2.NumberOfImages = i1.Count
	if i2.NumberOfImages <= 0 {
		i2.NumberOfImages = 1
	}
	if i2.NumberOfImages > 4 {
		i2.NumberOfImages = 4
	}

	i2.AspectRatio = i1.GeminiOptions.GetString("aspect_ratio")
	if i2.AspectRatio == "" {
		i2.AspectRatio = i1.GeminiOptions.GetString("aspectRatio")
	}
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

	i2.ResponseModalities = i1.GeminiOptions.GetStringArray("response_modalities")
	if len(i2.ResponseModalities) == 0 {
		i2.ResponseModalities = i1.GeminiOptions.GetStringArray("responseModalities")
	}
	i2.ImageSize = i1.GeminiOptions.GetString("image_size")
	if i2.ImageSize == "" {
		i2.ImageSize = i1.GeminiOptions.GetString("imageSize")
	}
}

func (i *GeminiCreateImagesInput) Verify() error {
	if strings.TrimSpace(i.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}

	switch i.Model {
	case GeminiModelImagen3:
		aspectRatio := []string{
			AspectRatioSquare,
			AspectRatioPortrait34,
			AspectRatioLandscape43,
			AspectRatioPortrait916,
			AspectRatioLandscape169,
		}
		personGeneration := []string{DontAllow, Allow}
		safetyFilterLevel := []string{BlockLowAndAbove, BlockMediumAndAbove, BlockOnlyHigh}
		if !slices.Contains(aspectRatio, i.AspectRatio) {
			return fmt.Errorf("aspect ratio must be one of %v", aspectRatio)
		}
		if !slices.Contains(personGeneration, i.PersonGeneration) {
			return fmt.Errorf("person generation must be one of %v", personGeneration)
		}
		if !slices.Contains(safetyFilterLevel, i.SafetyFilterLevel) {
			return fmt.Errorf("safety filter level must be one of %v", safetyFilterLevel)
		}
		return nil

	case GeminiModelNanoBanana, GeminiModelNanoBananaPro:
		aspectRatio := []string{
			AspectRatioSquare,
			AspectRatioPortrait23,
			AspectRatioLandscape32,
			AspectRatioPortrait34,
			AspectRatioLandscape43,
			AspectRatioPortrait45,
			AspectRatioLandscape54,
			AspectRatioPortrait916,
			AspectRatioLandscape169,
			AspectRatioLandscape219,
		}
		if i.AspectRatio != "" && !slices.Contains(aspectRatio, i.AspectRatio) {
			return fmt.Errorf("aspect ratio must be one of %v", aspectRatio)
		}

		if len(i.ResponseModalities) == 0 {
			i.ResponseModalities = []string{ResponseModalityText, ResponseModalityImage}
		}
		if err := verifyResponseModalities(i.ResponseModalities); err != nil {
			return err
		}

		if i.ImageSize != "" {
			allowed := []string{"1K", "2K", "4K"}
			if !slices.Contains(allowed, i.ImageSize) {
				return fmt.Errorf("image size must be one of %v", allowed)
			}
		}
		return nil
	default:
		return fmt.Errorf("%w: %s (supported: %s, %s, %s)", errInvalidGeminiModel, i.Model, GeminiModelImagen3, GeminiModelNanoBanana, GeminiModelNanoBananaPro)
	}
}

func GeminiCreateImages(ctx context.Context, token string, input *CreateImagesInput) (*CreateImagesOutput, error) {
	if input.Model == "" {
		input.Model = GeminiModelImagen3
	}

	geminiInput := &GeminiCreateImagesInput{}
	geminiInput.Loads(input)
	if err := geminiInput.Verify(); err != nil {
		return nil, err
	}

	switch geminiInput.Model {
	case GeminiModelImagen3:
		return geminiPredictImagen(ctx, token, geminiInput)
	case GeminiModelNanoBanana, GeminiModelNanoBananaPro:
		return geminiGenerateContentImages(ctx, token, geminiInput)
	default:
		return nil, fmt.Errorf("%w: %s", errInvalidGeminiModel, geminiInput.Model)
	}
}

func verifyResponseModalities(modalities []string) error {
	for _, m := range modalities {
		switch strings.ToUpper(strings.TrimSpace(m)) {
		case "TEXT", "IMAGE":
			// ok
		default:
			return fmt.Errorf("response modalities must be TEXT and/or IMAGE, got %q", m)
		}
	}
	return nil
}

func normalizeResponseModalities(modalities []string) []string {
	out := make([]string, 0, len(modalities))
	seen := map[string]bool{}
	for _, m := range modalities {
		m = strings.TrimSpace(m)
		switch strings.ToUpper(m) {
		case "TEXT":
			m = ResponseModalityText
		case "IMAGE":
			m = ResponseModalityImage
		default:
			// Pass through unknown strings (Verify() should have rejected).
		}
		if m == "" || seen[m] {
			continue
		}
		seen[m] = true
		out = append(out, m)
	}
	return out
}

func geminiPredictImagen(ctx context.Context, token string, geminiInput *GeminiCreateImagesInput) (*CreateImagesOutput, error) {
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

func geminiGenerateContentImages(ctx context.Context, token string, geminiInput *GeminiCreateImagesInput) (*CreateImagesOutput, error) {
	now := time.Now()
	result := &CreateImagesOutput{
		Created: int(now.Unix()),
	}

	modalities := normalizeResponseModalities(geminiInput.ResponseModalities)
	if len(modalities) == 0 {
		modalities = []string{ResponseModalityText, ResponseModalityImage}
	}

	for i := 0; i < geminiInput.NumberOfImages; i++ {
		resp, err := geminiGenerateContentOnce(ctx, token, geminiInput.Model, geminiInput.Prompt, modalities, geminiInput.AspectRatio, geminiInput.ImageSize)
		if err != nil {
			return nil, err
		}
		for _, item := range resp.images {
			result.Data = append(result.Data, struct {
				B64JSON string `json:"b64_json"`
			}{B64JSON: item.data})
			if result.MimeType == "" && item.mimeType != "" {
				result.MimeType = item.mimeType
			}
		}
	}

	if len(result.Data) == 0 {
		return nil, fmt.Errorf("no images returned from model %s", geminiInput.Model)
	}
	return result, nil
}

type geminiGeneratedImage struct {
	data     string
	mimeType string
}

type geminiGenerateContentParsed struct {
	text   string
	images []geminiGeneratedImage
}

func geminiGenerateContentOnce(ctx context.Context, token, model, prompt string, responseModalities []string, aspectRatio, imageSize string) (*geminiGenerateContentParsed, error) {
	reqBody := map[string]any{
		"contents": []map[string]any{
			{
				"parts": []map[string]any{
					{"text": prompt},
				},
			},
		},
	}

	genCfg := map[string]any{
		"responseModalities": responseModalities,
	}
	imgCfg := map[string]any{}
	if aspectRatio != "" {
		imgCfg["aspectRatio"] = aspectRatio
	}
	if imageSize != "" {
		imgCfg["imageSize"] = imageSize
	}
	if len(imgCfg) > 0 {
		genCfg["imageConfig"] = imgCfg
	}
	reqBody["generationConfig"] = genCfg

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, token)
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

	var data struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text       string `json:"text,omitempty"`
					InlineData struct {
						MimeType string `json:"mimeType,omitempty"`
						Data     string `json:"data,omitempty"`
					} `json:"inlineData,omitempty"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	parsed := &geminiGenerateContentParsed{}
	for _, cand := range data.Candidates {
		for _, part := range cand.Content.Parts {
			if part.Text != "" {
				parsed.text += part.Text
			}
			if part.InlineData.Data != "" {
				parsed.images = append(parsed.images, geminiGeneratedImage{
					data:     part.InlineData.Data,
					mimeType: part.InlineData.MimeType,
				})
			}
		}
	}

	return parsed, nil
}
