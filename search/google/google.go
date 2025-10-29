package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"log/slog"
)

const googleSearchAPI = "https://www.googleapis.com/customsearch/v1"

// SearchClient manages API interactions
type SearchClient struct {
	APIKey string
	CX     string
	Client *http.Client
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string         `json:"title"`
	Link    string         `json:"link"`
	Snippet string         `json:"snippet"`
	Pagemap map[string]any `json:"pagemap"`
}

// SearchResponse represents the API response
type SearchResponse struct {
	Items []SearchResult `json:"items"`
}

// NewSearchClient initializes a search client
func NewSearchClient(apiKey, cx string) *SearchClient {
	return &SearchClient{
		APIKey: apiKey,
		CX:     cx,
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

// Search executes a search query with pagination
func (s *SearchClient) Search(query string, start int, options ...map[string]string) (*SearchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	queryParams := url.Values{}
	queryParams.Set("key", s.APIKey)
	queryParams.Set("cx", s.CX)
	queryParams.Set("q", query)
	queryParams.Set("start", fmt.Sprintf("%d", start))
	// Append additional options if provided
	if len(options) > 0 {
		for k, v := range options[0] {
			queryParams.Set(k, v)
		}
	}

	url := fmt.Sprintf("%s?%s", googleSearchAPI, queryParams.Encode())

	slog.Info("Sending search request", "query", query, "start", start, "options", options)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		slog.Error("Failed to create request", "error", err)
		return nil, err
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		slog.Error("Request failed", "error", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		slog.Error("Failed to decode response", "error", err)
		return nil, err
	}

	slog.Info("Search request successful", "results", len(result.Items))
	return &result, nil
}
