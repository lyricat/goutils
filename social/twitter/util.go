package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

func (c *Client) getHTTPClient(ctx context.Context, token *oauth2.Token) *http.Client {
	if c.cfg.BearerToken != "" {
		return c.httpClient
	}
	return c.oauthConfig.Client(ctx, token)
}

func (c *Client) addAuthHeader(req *http.Request, token *oauth2.Token) {
	if c.cfg.BearerToken != "" {
		req.Header.Add("Authorization", "Bearer "+c.cfg.BearerToken)
	} else if token != nil {
		req.Header.Add("Authorization", "Bearer "+token.AccessToken)
	}
}

func (c *Client) ValidateToken(ctx context.Context, token *oauth2.Token) error {
	url := "https://api.x.com/2/users/me"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token validation failed: %s, body: %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) catchError(resp *http.Response, body []byte) error {
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Response Status: %s\n", resp.Status)
		fmt.Printf("Response Body: %s\n", string(body))

		var errorResponse struct {
			Errors []struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &errorResponse); err == nil && len(errorResponse.Errors) > 0 {
			return fmt.Errorf("twitter API error: %s (code: %d)", errorResponse.Errors[0].Message, errorResponse.Errors[0].Code)
		}
		return fmt.Errorf("failed to get tweets from list: %s, body: %s", resp.Status, string(body))
	}
	return nil
}
