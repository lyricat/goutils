package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"golang.org/x/oauth2"
)

func (c *Client) GetTweetByID(ctx context.Context, token *oauth2.Token, tweetID string) (*TweetResponse, error) {
	url := fmt.Sprintf("https://api.x.com/2/tweets/%s", tweetID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	q := req.URL.Query()
	q.Add("tweet.fields", "author_id,created_at,entities,public_metrics,referenced_tweets,lang")
	q.Add("expansions", "author_id,referenced_tweets.id")
	req.URL.RawQuery = q.Encode()

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error getting tweet: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if err := c.catchError(resp, body); err != nil {
		slog.Error("error getting tweets from list", "error", err)
		return nil, err
	}

	var result TweetResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}

func (c *Client) GetTweetsByIDs(ctx context.Context, token *oauth2.Token, tweetIDs []string) (*TweetsResponse, error) {
	url := "https://api.twitter.com/2/tweets"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	q := req.URL.Query()
	q.Add("ids", strings.Join(tweetIDs, ","))
	q.Add("tweet.fields", "author_id,created_at,entities,public_metrics,referenced_tweets,lang")
	q.Add("expansions", "author_id,referenced_tweets.id")
	req.URL.RawQuery = q.Encode()

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if err := c.catchError(resp, body); err != nil {
		slog.Error("error getting tweets from list", "error", err)
		return nil, err
	}

	var result TweetsResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}
