package speech

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type (
	Client struct {
		cfg Config
	}
	Config struct {
		AzureAPIKey   string
		AzureEndpoint string
	}
)

func New(cfg Config) *Client {
	return &Client{
		cfg: cfg,
	}
}

func (d *Client) ToText(langCode string, audioData []byte) (string, error) {
	url := fmt.Sprintf("%sspeech/recognition/conversation/cognitiveservices/v1?language=%s", d.cfg.AzureEndpoint, langCode)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(audioData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Ocp-Apim-Subscription-Key", d.cfg.AzureAPIKey)
	req.Header.Set("Content-Type", "audio/wav; codec=audio/pcm; samplerate=16000")

	client := &http.Client{Timeout: time.Second * 30}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if recognition, ok := result["DisplayText"].(string); ok {
		return recognition, nil
	}

	return "", fmt.Errorf("no recrecognitionognition result in response")
}
