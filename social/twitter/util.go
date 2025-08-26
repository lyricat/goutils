package twitter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
		return fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("token validation failed: %s, body: %s", resp.Status, string(body))
	}
	return nil
}

func (c *Client) catchError(resp *http.Response, body []byte) error {
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Response Status: %s\n", resp.Status)
		fmt.Printf("Response Body: %s\n", string(body))

		var errorResponse struct {
			Errors []struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &errorResponse); err == nil && len(errorResponse.Errors) > 0 {
			return fmt.Errorf("twitter API error: %s (code: %d)", errorResponse.Errors[0].Message, errorResponse.Errors[0].Code)
		}
		return fmt.Errorf("failed to get tweets from list: %s, body: %s", resp.Status, string(body))
	}
	return nil
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
