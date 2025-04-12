package twitter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		return nil, fmt.Errorf("failed to execute search request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Attempt to read error body for more details
		var apiError SearchResponse
		_ = json.NewDecoder(resp.Body).Decode(&apiError) // Ignore decode error here, focus on status
		errorMsg := fmt.Sprintf("twitter api error: status %s", resp.Status)
		if len(apiError.Errors) > 0 {
			errorMsg = fmt.Sprintf("%s - %s: %s", errorMsg, apiError.Errors[0].Title, apiError.Errors[0].Detail)
		} else if apiError.Errors != nil && apiError.Errors[0].Title != "" {
			// Handle cases where the top-level 'errors' field might exist but be empty,
			// or where a specific error structure is returned differently.
			// This part might need adjustment based on actual error responses observed.
			errorMsg = fmt.Sprintf("%s - Title: %s, Detail: %s", errorMsg, apiError.Errors[0].Title, apiError.Errors[0].Detail)
		}
		return nil, errors.New(errorMsg)
	}

	var searchResult SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResult); err != nil {
		return nil, fmt.Errorf("failed to decode search response: %w", err)
	}

	// Check for errors within the JSON response body itself
	if len(searchResult.Errors) > 0 {
		apiErr := searchResult.Errors[0]
		return nil, fmt.Errorf("twitter api error in response body - %s: %s (%s)", apiErr.Title, apiErr.Detail, apiErr.Type)
	}

	return &searchResult, nil
}
