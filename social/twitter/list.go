package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"golang.org/x/oauth2"
)

func (tr *TweetResponse) PrettyPrint() {
	// print the tweets in Data. If the referenced tweet is present, find it from includes.Tweets and  print that as well
	for _, tweet := range tr.Data {
		fmt.Printf("- Tweet ID: %s\n", tweet.ID)
		fmt.Printf("- Text: %s\n", tweet.Text)
		fmt.Printf("- Author ID: %s\n", tweet.AuthorID)
		if len(tweet.ReferencedTweets) > 0 {
			for _, rt := range tweet.ReferencedTweets {
				for _, t := range tr.Includes.Tweets {
					if t.ID == rt.ID {
						fmt.Printf("\t- Referenced Tweet Type: %s\n", rt.Type)
						fmt.Printf("\t- Referenced Tweet ID: %s\n", t.ID)
						fmt.Printf("\t- Referenced Tweet Text: %s\n", t.Text)
						fmt.Printf("\t- Referenced Tweet Author ID: %s\n", t.AuthorID)
					}
				}
			}
		}
		fmt.Printf("- Metrics: %+v\n", tweet.PublicMetrics)
		fmt.Printf("- Entities: %+v\n", tweet.Entities)
		fmt.Println()
	}
	fmt.Printf("Total Tweets: %d\n", len(tr.Data))
}

// GetTweetsFromList retrieves recent tweets from a given Twitter List
func (c *Client) GetTweetsFromList(ctx context.Context, token *oauth2.Token, listID string, maxResults int, paginationToken string) (*TweetResponse, error) {
	url := fmt.Sprintf("https://api.x.com/2/lists/%s/tweets", listID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	// includes links and quoted tweet and retweet information
	q.Add("tweet.fields", "author_id,created_at,entities,public_metrics,referenced_tweets,lang")
	q.Add("max_results", fmt.Sprintf("%d", maxResults))
	q.Add("expansions", "author_id,referenced_tweets.id")
	if paginationToken != "" {
		q.Add("pagination_token", paginationToken)
	}
	req.URL.RawQuery = q.Encode()

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("error getting tweets from list", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
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

// GetListMembers retrieves members of a Twitter List
func (c *Client) GetListMembers(ctx context.Context, token *oauth2.Token, listID string) (*ListMemberResponse, error) {
	url := fmt.Sprintf("https://api.x.com/2/lists/%s/members", listID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("error getting tweets from list", "error", err)
		return nil, err
	}

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		slog.Error("error getting tweets from list", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("error reading response body", "error", err)
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if err := c.catchError(resp, body); err != nil {
		slog.Error("error getting tweets from list", "error", err)
		return nil, err
	}

	var result ListMemberResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}
