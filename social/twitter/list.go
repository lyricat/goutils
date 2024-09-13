package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/oauth2"
)

type (
	TweetResponse struct {
		ID        string `json:"id"`
		Text      string `json:"text"`
		AuthorID  string `json:"author_id"`
		CreatedAt string `json:"created_at"`
	}
)

// GetTweetsFromList retrieves recent tweets from a given Twitter List
func (c *Client) GetTweetsFromList(ctx context.Context, token *oauth2.Token, listID string, maxResults int) ([]TweetResponse, error) {
	url := fmt.Sprintf("https://api.twitter.com/2/lists/%s/tweets", listID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("tweet.fields", "author_id,created_at")
	q.Add("max_results", fmt.Sprintf("%d", maxResults))
	req.URL.RawQuery = q.Encode()

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Errors []struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &errorResponse); err == nil && len(errorResponse.Errors) > 0 {
			return nil, fmt.Errorf("twitter API error: %s (code: %d)", errorResponse.Errors[0].Message, errorResponse.Errors[0].Code)
		}
		return nil, fmt.Errorf("failed to get tweets from list: %s, body: %s", resp.Status, string(body))
	}

	var result struct {
		Data []TweetResponse `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return result.Data, nil
}
