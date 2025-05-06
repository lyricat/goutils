package bluesky

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type BlueskyClient struct {
	username    string
	appPassword string
	serviceUrl  string
	accessJwt   string
	did         string
}

func New(
	serviceUrl string,
	username string,
	appPassword string,
) *BlueskyClient {
	return &BlueskyClient{
		username:    username,
		appPassword: appPassword,
		serviceUrl:  serviceUrl,
	}
}

func (c *BlueskyClient) Authenticate() error {
	if c.username == "" || c.appPassword == "" {
		return fmt.Errorf("missing Bluesky credentials")
	}

	if c.serviceUrl == "" {
		c.serviceUrl = "https://bsky.social"
	}

	authData := map[string]string{
		"identifier": c.username,
		"password":   c.appPassword,
	}

	jsonData, err := json.Marshal(authData)
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %v", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/xrpc/com.atproto.server.createSession", c.serviceUrl), bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("authentication failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status code %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode response: %v", err)
	}

	accessJwt, ok := result["accessJwt"].(string)
	if !ok {
		return fmt.Errorf("access token not found in response")
	}

	did, ok := result["did"].(string)
	if !ok {
		return fmt.Errorf("DID not found in response")
	}

	// Store for later use
	c.accessJwt = accessJwt
	c.did = did

	return nil
}
