package twitter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strings"
	"time"

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
		return wrapAPIRequestError(req, "error sending request", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}

	if err := c.checkAPIResponse(req, resp, body, http.StatusOK); err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}
	return nil
}

func (c *Client) checkAPIResponse(req *http.Request, resp *http.Response, body []byte, okStatusCodes ...int) error {
	if statusAllowed(resp.StatusCode, okStatusCodes...) {
		return nil
	}

	err := buildAPIResponseError(req, resp, body)
	logAPIResponseError(req, resp, body, err)
	return err
}

func statusAllowed(statusCode int, okStatusCodes ...int) bool {
	for _, okStatusCode := range okStatusCodes {
		if statusCode == okStatusCode {
			return true
		}
	}
	return false
}

func wrapAPIRequestError(req *http.Request, message string, err error) error {
	attrs := []any{"message", message}
	if req == nil {
		wrappedErr := fmt.Errorf("%s: %w", message, err)
		attrs = append(attrs, "error", wrappedErr)
		slog.Error("x api request execution failed", attrs...)
		return wrappedErr
	}
	wrappedErr := fmt.Errorf("%s: %s %s: %w", message, req.Method, req.URL.String(), err)
	attrs = append(attrs,
		"error", wrappedErr,
		"method", req.Method,
		"url", req.URL.String(),
	)
	slog.Error("x api request execution failed", attrs...)
	return wrappedErr
}

func buildAPIResponseError(req *http.Request, resp *http.Response, body []byte) error {
	method := "<nil>"
	requestURL := "<nil>"
	if req != nil {
		method = req.Method
		requestURL = req.URL.String()
	}

	details := extractAPIErrorDetails(body)
	messageParts := []string{fmt.Sprintf("%s %s returned %s", method, requestURL, resp.Status)}
	if details != "" {
		messageParts = append(messageParts, details)
	}
	messageParts = append(messageParts, fmt.Sprintf("body: %s", string(body)))

	if rateLimit := formatRateLimitHeaders(resp); rateLimit != "" {
		messageParts = append(messageParts, rateLimit)
	}

	return errors.New(strings.Join(messageParts, ", "))
}

func logAPIResponseError(req *http.Request, resp *http.Response, body []byte, err error) {
	attrs := []any{
		"error", err,
		"status", resp.Status,
		"status_code", resp.StatusCode,
		"body", string(body),
	}
	if req != nil {
		attrs = append(attrs,
			"method", req.Method,
			"url", req.URL.String(),
		)
	}
	if limit := resp.Header.Get("x-rate-limit-limit"); limit != "" {
		attrs = append(attrs, "x_rate_limit_limit", limit)
	}
	if remaining := resp.Header.Get("x-rate-limit-remaining"); remaining != "" {
		attrs = append(attrs, "x_rate_limit_remaining", remaining)
	}
	if reset := resp.Header.Get("x-rate-limit-reset"); reset != "" {
		attrs = append(attrs, "x_rate_limit_reset", reset)
	}

	slog.Error("x api request failed", attrs...)
}

func extractAPIErrorDetails(body []byte) string {
	var errorResponse struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
		Type   string `json:"type"`
		Status int    `json:"status"`
		Errors []struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
			Title   string `json:"title"`
			Detail  string `json:"detail"`
			Type    string `json:"type"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(body, &errorResponse); err != nil {
		return ""
	}

	if len(errorResponse.Errors) > 0 {
		first := errorResponse.Errors[0]
		switch {
		case first.Message != "" && first.Code != 0:
			return fmt.Sprintf("api_error: %s (code: %d)", first.Message, first.Code)
		case first.Message != "":
			return fmt.Sprintf("api_error: %s", first.Message)
		case first.Title != "" || first.Detail != "":
			return fmt.Sprintf("api_error: %s - %s (%s)", first.Title, first.Detail, first.Type)
		}
	}
	if errorResponse.Title != "" || errorResponse.Detail != "" {
		return fmt.Sprintf("api_error: %s - %s (%s)", errorResponse.Title, errorResponse.Detail, errorResponse.Type)
	}
	return ""
}

func formatRateLimitHeaders(resp *http.Response) string {
	parts := []string{}
	if limit := resp.Header.Get("x-rate-limit-limit"); limit != "" {
		parts = append(parts, fmt.Sprintf("x-rate-limit-limit=%s", limit))
	}
	if remaining := resp.Header.Get("x-rate-limit-remaining"); remaining != "" {
		parts = append(parts, fmt.Sprintf("x-rate-limit-remaining=%s", remaining))
	}
	if reset := resp.Header.Get("x-rate-limit-reset"); reset != "" {
		parts = append(parts, fmt.Sprintf("x-rate-limit-reset=%s", reset))
	}
	return strings.Join(parts, " ")
}

func FetchWebpageMetadata(url string) (string, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", "", fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; TwitterBot/1.0)")

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("error fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("HTTP error: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", fmt.Errorf("error reading response body: %w", err)
	}

	html := string(body)
	title := extractTitle(html)
	description := extractDescription(html)

	return title, description, nil
}

func extractTitle(html string) string {
	// Try Open Graph title first
	ogTitleRegex := regexp.MustCompile(`(?i)<meta[^>]*property\s*=\s*["']og:title["'][^>]*content\s*=\s*["']([^"']+)["']`)
	matches := ogTitleRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return cleanText(matches[1])
	}

	// Try Twitter Card title
	twitterTitleRegex := regexp.MustCompile(`(?i)<meta[^>]*name\s*=\s*["']twitter:title["'][^>]*content\s*=\s*["']([^"']+)["']`)
	matches = twitterTitleRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return cleanText(matches[1])
	}

	// Fallback to HTML title tag
	titleRegex := regexp.MustCompile(`(?i)<title[^>]*>([^<]+)</title>`)
	matches = titleRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return cleanText(matches[1])
	}

	return ""
}

func extractDescription(html string) string {
	// Try Open Graph description first
	ogDescRegex := regexp.MustCompile(`(?i)<meta[^>]*property\s*=\s*["']og:description["'][^>]*content\s*=\s*["']([^"']+)["']`)
	matches := ogDescRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return cleanText(matches[1])
	}

	// Try Twitter Card description
	twitterDescRegex := regexp.MustCompile(`(?i)<meta[^>]*name\s*=\s*["']twitter:description["'][^>]*content\s*=\s*["']([^"']+)["']`)
	matches = twitterDescRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return cleanText(matches[1])
	}

	// Try standard meta description
	descRegex := regexp.MustCompile(`(?i)<meta[^>]*name\s*=\s*["']description["'][^>]*content\s*=\s*["']([^"']+)["']`)
	matches = descRegex.FindStringSubmatch(html)
	if len(matches) > 1 {
		return cleanText(matches[1])
	}

	return ""
}

func cleanText(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")
	text = strings.ReplaceAll(text, "\t", " ")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return text
}

func (t *TweetObject) GetExpandedURLs() []URLInfo {
	var urlInfos []URLInfo
	for _, url := range t.Entities.Urls {
		// Only include URLs that Twitter has expanded (not just t.co links)
		if url.ExpandedURL != "" && url.ExpandedURL != url.URL {
			urlInfo := URLInfo{
				URL:         url.URL,
				ExpandedURL: url.ExpandedURL,
				DisplayURL:  url.DisplayURL,
			}
			urlInfos = append(urlInfos, urlInfo)
		}
	}
	return urlInfos
}

func (t *TweetObject) GetExpandedURLsWithMetadata() []URLInfo {
	var urlInfos []URLInfo
	for _, url := range t.Entities.Urls {
		// Only include URLs that Twitter has expanded (not just t.co links)
		if url.ExpandedURL != "" && url.ExpandedURL != url.URL {
			urlInfo := URLInfo{
				URL:         url.URL,
				ExpandedURL: url.ExpandedURL,
				DisplayURL:  url.DisplayURL,
			}

			// Fetch metadata from the actual webpage
			if title, description, err := FetchWebpageMetadata(url.ExpandedURL); err == nil {
				urlInfo.Title = title
				urlInfo.Description = description
			}

			urlInfos = append(urlInfos, urlInfo)
		}
	}
	return urlInfos
}
