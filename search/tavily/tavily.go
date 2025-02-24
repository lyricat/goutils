package tavily

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// SearchRequest represents the request body for the search API.
type SearchRequest struct {
	Query                    string   `json:"query"`
	Topic                    string   `json:"topic"`
	SearchDepth              string   `json:"search_depth"`
	MaxResults               int      `json:"max_results"`
	TimeRange                *string  `json:"time_range"`
	Days                     int      `json:"days"`
	IncludeAnswer            bool     `json:"include_answer"`
	IncludeRawContent        bool     `json:"include_raw_content"`
	IncludeImages            bool     `json:"include_images"`
	IncludeImageDescriptions bool     `json:"include_image_descriptions"`
	IncludeDomains           []string `json:"include_domains"`
	ExcludeDomains           []string `json:"exclude_domains"`
	Start                    int      `json:"start"`
}

// SearchResult represents an individual result in the search response.
type SearchResult struct {
	Title      string  `json:"title"`
	URL        string  `json:"url"`
	Content    string  `json:"content"`
	Score      float64 `json:"score"`
	RawContent *string `json:"raw_content"`
}

// SearchResponse represents the successful response from the search API.
type SearchResponse struct {
	Query        string         `json:"query"`
	Answer       string         `json:"answer"`
	Images       []string       `json:"images"`
	Results      []SearchResult `json:"results"`
	ResponseTime float64        `json:"response_time"`
}

// SearchClient holds the API key and HTTP client for making requests.
type SearchClient struct {
	apiKey string
	client *http.Client
}

// NewSearchClient creates a new SearchClient with the given API key.
func NewSearchClient(apiKey string) *SearchClient {
	return &SearchClient{
		apiKey: apiKey,
		client: &http.Client{},
	}
}

// Search performs a search query using the provided query and start parameters.
// It returns a SearchResponse on success or an error if the request fails.
func (s *SearchClient) Search(query string, start int) (*SearchResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set default values for the request
	reqBody := SearchRequest{
		Query:                    query,
		Topic:                    "general",
		SearchDepth:              "basic",
		MaxResults:               10,
		TimeRange:                nil,
		Days:                     3,
		IncludeAnswer:            true,
		IncludeRawContent:        false,
		IncludeImages:            false,
		IncludeImageDescriptions: false,
		IncludeDomains:           []string{},
		ExcludeDomains:           []string{},
		Start:                    start,
	}

	// Marshal the request body to JSON
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	slog.Info("[goutils.search] Preparing search request", "query", query, "start", start)
	// Create the HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.tavily.com/search", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	slog.Info("[goutils.search] Sending search request", "query", query, "start", start)
	// Send the request
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Check the status code
	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("API returned non-200 status code (%d) and failed to read error body: %w", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Unmarshal the response
	var searchResp SearchResponse
	err = json.NewDecoder(resp.Body).Decode(&searchResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return &searchResp, nil
}
