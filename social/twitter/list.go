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

func (tr *TweetsResponse) PrettyPrint(w io.Writer) {
	if w == nil {
		w = io.Discard
	}
	// print the tweets in Data. If the referenced tweet is present, find it from includes.Tweets and  print that as well
	for _, tweet := range tr.Data {
		fmt.Fprintf(w, "- Tweet ID: %s\n", tweet.ID)
		fmt.Fprintf(w, "- Text: %s\n", tweet.Text)
		fmt.Fprintf(w, "- Author ID: %s\n", tweet.AuthorID)
		if len(tweet.ReferencedTweets) > 0 {
			for _, rt := range tweet.ReferencedTweets {
				for _, t := range tr.Includes.Tweets {
					if t.ID == rt.ID {
						fmt.Fprintf(w, "\t- Referenced Tweet Type: %s\n", rt.Type)
						fmt.Fprintf(w, "\t- Referenced Tweet ID: %s\n", t.ID)
						fmt.Fprintf(w, "\t- Referenced Tweet Text: %s\n", t.Text)
						fmt.Fprintf(w, "\t- Referenced Tweet Author ID: %s\n", t.AuthorID)
					}
				}
			}
		}
		fmt.Fprintf(w, "- Metrics: %+v\n", tweet.PublicMetrics)
		fmt.Fprintf(w, "- Entities: %+v\n", tweet.Entities)
		fmt.Fprintln(w)
	}
	fmt.Fprintf(w, "Total Tweets: %d\n", len(tr.Data))
}

// GetTweetsFromList retrieves recent tweets from a given Twitter List
func (c *Client) GetTweetsFromList(ctx context.Context, token *oauth2.Token, listID string, maxResults int, paginationToken string) (*TweetsResponse, error) {
	url := fmt.Sprintf("https://api.x.com/2/lists/%s/tweets", listID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	// includes links and quoted tweet and retweet information
	q.Add("tweet.fields", "author_id,created_at,entities,public_metrics,referenced_tweets,lang")
	q.Add("user.fields", "id,name,profile_image_url,username,public_metrics")
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
		return nil, wrapAPIRequestError(req, "error getting tweets from list", err)
	}
	defer resp.Body.Close()

	// Read the response body
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

// GetListMembers retrieves members of a Twitter List
func (c *Client) GetListMembers(ctx context.Context, token *oauth2.Token, listID string, maxResults int, paginationToken string) (*ListMemberResponse, error) {
	url := fmt.Sprintf("https://api.x.com/2/lists/%s/members", listID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		slog.Error("error getting members from list", "error", err)
		return nil, err
	}

	q := req.URL.Query()
	// includes links and quoted tweet and retweet information
	q.Add("user.fields", "id,name,profile_image_url,username,public_metrics")
	q.Add("max_results", fmt.Sprintf("%d", maxResults))
	if paginationToken != "" {
		q.Add("pagination_token", paginationToken)
	}

	req.URL.RawQuery = q.Encode()

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, wrapAPIRequestError(req, "error getting members from list", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	if err := c.checkAPIResponse(req, resp, body, http.StatusOK); err != nil {
		return nil, err
	}

	var result ListMemberResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}
