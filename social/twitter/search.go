package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

// SearchTweets performs a search query against the recent search endpoint.
// It requires a user context token.
// query: The search query string (see Twitter API docs for syntax).
// nextToken: Optional token for pagination (get next page of results).
func (c *Client) SearchTweets(ctx context.Context, token *oauth2.Token, query, nextToken string, limit int) (*SearchResponse, error) {
	endpoint := "https://api.x.com/2/tweets/search/recent"
	params := url.Values{}
	params.Set("query", query)
	params.Set("tweet.fields", "created_at,public_metrics,author_id") // Specify desired tweet fields
	params.Set("expansions", "author_id")                             // Expand author details
	params.Set("user.fields", "profile_image_url,username,name")      // Specify desired user fields
	params.Set("max_results", fmt.Sprintf("%d", limit))               // Adjust as needed (max 100 for recent search)

	if nextToken != "" {
		params.Set("next_token", nextToken)
	}

	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create search request: %w", err)
	}

	c.addAuthHeader(req, token)

	client := c.getHTTPClient(ctx, token)
	resp, err := client.Do(req)
	if err != nil {
		return nil, wrapAPIRequestError(req, "failed to execute search request", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response: %w", err)
	}

	if err := c.checkAPIResponse(req, resp, body, http.StatusOK); err != nil {
		return nil, err
	}

	var searchResult SearchResponse
	if err := json.Unmarshal(body, &searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Check for errors within the JSON response body itself
	if len(searchResult.Errors) > 0 {
		apiErr := searchResult.Errors[0]
		err := fmt.Errorf("%s %s returned API errors in response body: %s: %s (%s)", req.Method, req.URL.String(), apiErr.Title, apiErr.Detail, apiErr.Type)
		slog.Error("x api response contained errors", "error", err, "method", req.Method, "url", req.URL.String())
		return nil, err
	}

	return &searchResult, nil
}
