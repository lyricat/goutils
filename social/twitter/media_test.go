package twitter

import (
	"context"
	"os"
	"testing"

	"golang.org/x/oauth2"
)

func TestTweetMediaAndAltText(t *testing.T) {
	bearerToken := os.Getenv("TWITTER_BEARER_TOKEN")
	if bearerToken == "" {
		t.Skip("TWITTER_BEARER_TOKEN environment variable not set")
	}

	cfg := Config{
		BearerToken: bearerToken,
	}
	client := New(cfg, nil) // Pass nil for Redis client since we don't need it for this test
	token := &oauth2.Token{
		AccessToken: bearerToken,
		TokenType:   "Bearer",
	}

	// Test with a tweet that has images with alt text
	// Using a known tweet with images - you may need to replace this with an actual tweet ID
	tweetID := "1769997593105592483" // Replace with actual tweet ID that has images

	t.Run("GetTweetByID_WithMedia", func(t *testing.T) {
		response, err := client.GetTweetByID(context.Background(), token, tweetID)
		if err != nil {
			t.Logf("Error getting tweet (this might be expected if tweet doesn't exist): %v", err)
			return
		}

		tweet := response.Data
		t.Logf("Tweet text: %s", tweet.Text)

		if tweet.HasMedia() {
			t.Logf("Tweet has %d media attachments", len(tweet.Attachments.MediaKeys))

			mediaList := tweet.GetAllMediaWithAltText(response)
			for i, media := range mediaList {
				t.Logf("Media %d:", i+1)
				t.Logf("  Type: %s", media.Type)
				t.Logf("  URL: %s", media.URL)
				t.Logf("  Alt text: %s", media.AltText)
				t.Logf("  Dimensions: %dx%d", media.Width, media.Height)
			}
		} else {
			t.Log("Tweet has no media attachments")
		}
	})

	t.Run("GetTweetsByIDs_WithMedia", func(t *testing.T) {
		tweetIDs := []string{tweetID}
		response, err := client.GetTweetsByIDs(context.Background(), token, tweetIDs)
		if err != nil {
			t.Logf("Error getting tweets (this might be expected if tweets don't exist): %v", err)
			return
		}

		if len(response.Data) == 0 {
			t.Log("No tweets returned")
			return
		}

		tweet := response.Data[0]
		t.Logf("Tweet text: %s", tweet.Text)

		if tweet.HasMedia() {
			t.Logf("Tweet has %d media attachments", len(tweet.Attachments.MediaKeys))

			mediaList := tweet.GetAllMediaWithAltText(response)
			for i, media := range mediaList {
				t.Logf("Media %d:", i+1)
				t.Logf("  Type: %s", media.Type)
				t.Logf("  URL: %s", media.URL)
				t.Logf("  Alt text: %s", media.AltText)
			}
		} else {
			t.Log("Tweet has no media attachments")
		}

		// Test URL functionality
		if tweet.HasURL() {
			t.Logf("Tweet has %d URLs", len(tweet.Entities.Urls))
			
			// Test basic URL info (no HTTP requests)
			expandedURLs := tweet.GetExpandedURLs()
			for i, urlInfo := range expandedURLs {
				t.Logf("URL %d:", i+1)
				t.Logf("  Short URL: %s", urlInfo.URL)
				t.Logf("  Expanded URL: %s", urlInfo.ExpandedURL)
				t.Logf("  Display URL: %s", urlInfo.DisplayURL)
			}

			// Test URL metadata fetching (makes HTTP requests)
			t.Log("Fetching URL metadata...")
			urlsWithMetadata := tweet.GetExpandedURLsWithMetadata()
			for i, urlInfo := range urlsWithMetadata {
				t.Logf("URL %d with metadata:", i+1)
				t.Logf("  URL: %s", urlInfo.ExpandedURL)
				t.Logf("  Title: %s", urlInfo.Title)
				t.Logf("  Description: %s", urlInfo.Description)
			}
		} else {
			t.Log("Tweet has no URLs")
		}
	})
}

func TestMediaHelperMethods(t *testing.T) {
	// Test helper methods with mock data
	tweet := TweetObject{
		ID:   "123",
		Text: "Test tweet with media",
		Attachments: Attachments{
			MediaKeys: []string{"media1", "media2"},
		},
	}

	response := &TweetResponse{
		Data: tweet,
		Includes: struct {
			Users  []User        `json:"users"`
			Tweets []TweetObject `json:"tweets"`
			Media  []Media       `json:"media"`
		}{
			Media: []Media{
				{
					MediaKey: "media1",
					Type:     "photo",
					URL:      "https://example.com/image1.jpg",
					AltText:  "A beautiful sunset",
					Width:    1200,
					Height:   800,
				},
				{
					MediaKey: "media2",
					Type:     "photo",
					URL:      "https://example.com/image2.jpg",
					AltText:  "A mountain landscape",
					Width:    1920,
					Height:   1080,
				},
			},
		},
	}

	t.Run("HasMedia", func(t *testing.T) {
		if !tweet.HasMedia() {
			t.Error("Expected tweet to have media")
		}
	})

	t.Run("GetMediaKeys", func(t *testing.T) {
		keys := tweet.GetMediaKeys()
		if len(keys) != 2 {
			t.Errorf("Expected 2 media keys, got %d", len(keys))
		}
		if keys[0] != "media1" || keys[1] != "media2" {
			t.Errorf("Unexpected media keys: %v", keys)
		}
	})

	t.Run("GetMediaByKey", func(t *testing.T) {
		media := response.GetMediaByKey("media1")
		if media == nil {
			t.Fatal("Expected to find media1")
		}
		if media.AltText != "A beautiful sunset" {
			t.Errorf("Expected alt text 'A beautiful sunset', got '%s'", media.AltText)
		}
		if media.Type != "photo" {
			t.Errorf("Expected type 'photo', got '%s'", media.Type)
		}
	})

	t.Run("GetAllMediaWithAltText", func(t *testing.T) {
		mediaList := tweet.GetAllMediaWithAltText(response)
		if len(mediaList) != 2 {
			t.Errorf("Expected 2 media items, got %d", len(mediaList))
		}

		// Check first media
		if mediaList[0].AltText != "A beautiful sunset" {
			t.Errorf("Expected first media alt text 'A beautiful sunset', got '%s'", mediaList[0].AltText)
		}

		// Check second media
		if mediaList[1].AltText != "A mountain landscape" {
			t.Errorf("Expected second media alt text 'A mountain landscape', got '%s'", mediaList[1].AltText)
		}
	})

	t.Run("NoMedia", func(t *testing.T) {
		emptyTweet := TweetObject{
			ID:   "456",
			Text: "Tweet without media",
		}

		if emptyTweet.HasMedia() {
			t.Error("Expected tweet to have no media")
		}

		mediaList := emptyTweet.GetAllMediaWithAltText(response)
		if len(mediaList) != 0 {
			t.Errorf("Expected no media, got %d items", len(mediaList))
		}
	})
}

func TestURLHelperMethods(t *testing.T) {
	// Test tweet with URLs
	tweet := TweetObject{
		ID:   "123",
		Text: "Check out this link https://t.co/abc123 and this one https://t.co/def456",
		Entities: TweetEntities{
			Urls: []struct {
				URL         string `json:"url"`
				ExpandedURL string `json:"expanded_url"`
				DisplayURL  string `json:"display_url"`
			}{
				{
					URL:         "https://t.co/abc123",
					ExpandedURL: "https://example.com/article",
					DisplayURL:  "example.com/article",
				},
				{
					URL:         "https://t.co/def456",
					ExpandedURL: "https://github.com/user/repo",
					DisplayURL:  "github.com/user/repo",
				},
			},
		},
	}

	t.Run("HasURL", func(t *testing.T) {
		if !tweet.HasURL() {
			t.Error("Expected tweet to have URLs")
		}
	})

	t.Run("GetExpandedURLs", func(t *testing.T) {
		urlInfos := tweet.GetExpandedURLs()
		if len(urlInfos) != 2 {
			t.Errorf("Expected 2 URL infos, got %d", len(urlInfos))
		}

		// Check first URL
		if urlInfos[0].URL != "https://t.co/abc123" {
			t.Errorf("Expected first URL 'https://t.co/abc123', got '%s'", urlInfos[0].URL)
		}
		if urlInfos[0].ExpandedURL != "https://example.com/article" {
			t.Errorf("Expected first expanded URL 'https://example.com/article', got '%s'", urlInfos[0].ExpandedURL)
		}
		if urlInfos[0].DisplayURL != "example.com/article" {
			t.Errorf("Expected first display URL 'example.com/article', got '%s'", urlInfos[0].DisplayURL)
		}

		// Check second URL
		if urlInfos[1].ExpandedURL != "https://github.com/user/repo" {
			t.Errorf("Expected second expanded URL 'https://github.com/user/repo', got '%s'", urlInfos[1].ExpandedURL)
		}
	})

	t.Run("NoExpandedURLs", func(t *testing.T) {
		// Test tweet with no expanded URLs (e.g., just t.co links without expansion)
		tweetNoExpanded := TweetObject{
			ID:   "456",
			Text: "Just a t.co link",
			Entities: TweetEntities{
				Urls: []struct {
					URL         string `json:"url"`
					ExpandedURL string `json:"expanded_url"`
					DisplayURL  string `json:"display_url"`
				}{
					{
						URL:         "https://t.co/xyz789",
						ExpandedURL: "https://t.co/xyz789", // Same as URL, so not expanded
						DisplayURL:  "t.co/xyz789",
					},
				},
			},
		}

		urlInfos := tweetNoExpanded.GetExpandedURLs()
		if len(urlInfos) != 0 {
			t.Errorf("Expected no expanded URLs, got %d", len(urlInfos))
		}
	})

	t.Run("NoURLs", func(t *testing.T) {
		emptyTweet := TweetObject{
			ID:   "789",
			Text: "Tweet without URLs",
		}

		if emptyTweet.HasURL() {
			t.Error("Expected tweet to have no URLs")
		}

		urlInfos := emptyTweet.GetExpandedURLs()
		if len(urlInfos) != 0 {
			t.Errorf("Expected no URLs, got %d items", len(urlInfos))
		}
	})
}

func TestURLMetadataFetching(t *testing.T) {
	bearerToken := os.Getenv("TWITTER_BEARER_TOKEN")
	if bearerToken == "" {
		t.Skip("TWITTER_BEARER_TOKEN environment variable not set")
	}

	cfg := Config{
		BearerToken: bearerToken,
	}
	client := New(cfg, nil)
	token := &oauth2.Token{
		AccessToken: bearerToken,
		TokenType:   "Bearer",
	}

	// Use the specific tweet ID that likely has URLs
	tweetID := "1960143586592489815"

	t.Run("RealTweetURLMetadata", func(t *testing.T) {
		response, err := client.GetTweetByID(context.Background(), token, tweetID)
		if err != nil {
			t.Logf("Error getting tweet: %v", err)
			return
		}

		tweet := response.Data
		t.Logf("Tweet text: %s", tweet.Text)

		if tweet.HasURL() {
			t.Logf("Tweet has %d URLs", len(tweet.Entities.Urls))
			
			// Test URL metadata fetching (makes HTTP requests)
			t.Log("Fetching URL metadata...")
			urlsWithMetadata := tweet.GetExpandedURLsWithMetadata()
			for i, urlInfo := range urlsWithMetadata {
				t.Logf("URL %d with metadata:", i+1)
				t.Logf("  Short URL: %s", urlInfo.URL)
				t.Logf("  Expanded URL: %s", urlInfo.ExpandedURL)
				t.Logf("  Display URL: %s", urlInfo.DisplayURL)
				t.Logf("  Title: %s", urlInfo.Title)
				t.Logf("  Description: %s", urlInfo.Description)
			}
		} else {
			t.Log("Tweet has no URLs")
		}
	})

	t.Run("FetchWebpageMetadata", func(t *testing.T) {
		// Test with example.com
		title, description, err := FetchWebpageMetadata("https://example.com")
		if err != nil {
			t.Logf("Error fetching metadata: %v", err)
		} else {
			t.Logf("Example.com - Title: %s", title)
			t.Logf("Example.com - Description: %s", description)
		}
	})
}
