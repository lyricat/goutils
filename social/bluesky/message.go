package bluesky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type (
	Embed struct {
		Type     string         `json:"$type"`
		Images   *EmbedImage    `json:"images,omitempty"`
		External *EmbedExternal `json:"external,omitempty"`
	}

	EmbedExternal struct {
		Uri         string    `json:"uri"`
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Thumb       EmbedBlob `json:"thumb"`
	}

	EmbedImage struct {
		Images      EmbedBlob `json:"images"`
		Alt         string    `json:"alt"`
		AspectRatio struct {
			Width  int `json:"width"`
			Height int `json:"height"`
		} `json:"aspectRatio"`
	}

	EmbedBlob struct {
		Type string `json:"$type"`
		Ref  struct {
			Link string `json:"$link"`
		} `json:"ref"`
		MimeType string `json:"mimeType"`
		Size     int    `json:"size"`
	}
)

func (c *BlueskyClient) Post(text, lang string, embed Embed) (string, error) {
	if c.accessJwt == "" || c.did == "" {
		if err := c.Authenticate(); err != nil {
			return "", err
		}
	}

	record := map[string]interface{}{
		"$type":     "app.bsky.feed.post",
		"text":      text,
		"createdAt": time.Now().UTC().Format(time.RFC3339),
		"langs":     []string{lang},
		"embed":     embed,
	}

	postData := map[string]interface{}{
		"repo":       c.did,
		"collection": "app.bsky.feed.post",
		"record":     record,
	}

	jsonData, err := json.Marshal(postData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal post data: %v", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/xrpc/com.atproto.repo.createRecord", c.serviceUrl), bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.accessJwt)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create post: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create post with status code %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	uri, ok := result["uri"].(string)
	if !ok {
		return "", fmt.Errorf("post URI not found in response")
	}

	return uri, nil
}
