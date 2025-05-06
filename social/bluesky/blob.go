package bluesky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func (c *BlueskyClient) UploadBlob(url string) (*EmbedBlob, error) {
	if c.accessJwt == "" || c.did == "" {
		if err := c.Authenticate(); err != nil {
			return nil, err
		}
	}

	// get url content
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("failed to get url content: %d", resp.StatusCode)
	}

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}

	client := &http.Client{}

	uploadEP := c.serviceUrl + "/xrpc/com.atproto.repo.uploadBlob"
	req, err := http.NewRequest(http.MethodPost, uploadEP, bytes.NewReader(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", resp.Header.Get("Content-Type"))
	req.Header.Set("Authorization", "Bearer "+c.accessJwt)

	resp, err = client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode/100 != 2 {
		// read body as string
		buf, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("upload blob: non-200 status %d, %s", resp.StatusCode, string(buf))
	}
	var parsed struct {
		Blob EmbedBlob `json:"blob"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	return &parsed.Blob, nil
}
