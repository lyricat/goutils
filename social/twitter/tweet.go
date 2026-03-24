package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	q.Add("tweet.fields", "author_id,created_at,entities,public_metrics,referenced_tweets,lang,attachments")
	q.Add("user.fields", "id,name,profile_image_url,username,public_metrics")
	q.Add("media.fields", "media_key,type,url,alt_text,duration_ms,height,preview_image_url,public_metrics,width")
	q.Add("expansions", "author_id,referenced_tweets.id,attachments.media_keys")
	req.URL.RawQuery = q.Encode()

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, wrapAPIRequestError(req, "error getting tweet", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if err := c.checkAPIResponse(req, resp, body, http.StatusOK); err != nil {
		return nil, err
	}

	var result TweetResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}

func (c *Client) GetTweetsByIDs(ctx context.Context, token *oauth2.Token, tweetIDs []string) (*TweetsResponse, error) {
	url := "https://api.x.com/2/tweets"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	q := req.URL.Query()
	q.Add("ids", strings.Join(tweetIDs, ","))
	// Workaround: X intermittently returns 503 for large GetTweetsByIDs batches when
	// entities is requested. The current feedstream updater caller only needs tweet
	// text/timestamps/public_metrics for score refresh, so omit entities here.
	q.Add("tweet.fields", "author_id,created_at,public_metrics,referenced_tweets,lang,attachments")
	q.Add("user.fields", "id,name,profile_image_url,username,public_metrics")
	q.Add("media.fields", "media_key,type,url,alt_text,duration_ms,height,preview_image_url,public_metrics,width")
	q.Add("expansions", "author_id,referenced_tweets.id,attachments.media_keys")
	req.URL.RawQuery = q.Encode()

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, wrapAPIRequestError(req, "error sending request", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if err := c.checkAPIResponse(req, resp, body, http.StatusOK); err != nil {
		return nil, err
	}

	var result TweetsResponse

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}
