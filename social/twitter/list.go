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

type (
	TweetEntities struct {
		Urls []struct {
			URL         string `json:"url"`
			ExpandedURL string `json:"expanded_url"`
			DisplayURL  string `json:"display_url"`
		} `json:"urls"`
		Hashtags []struct {
			Tag   string `json:"tag"`
			Start int    `json:"start"`
			End   int    `json:"end"`
		} `json:"hashtags"`
		Mentions []struct {
			Username string `json:"username"`
			Start    int    `json:"start"`
			End      int    `json:"end"`
			ID       string `json:"id"`
		} `json:"mentions"`
		Annotations []struct {
			Start          int     `json:"start"`
			End            int     `json:"end"`
			Probability    float64 `json:"probability"`
			Type           string  `json:"type"`
			NormalizedText string  `json:"normalized_text"`
		} `json:"annotations"`
		CashTags []struct {
			Start int    `json:"start"`
			End   int    `json:"end"`
			Tag   string `json:"tag"`
		} `json:"cashtags"`
	}
	TweetPublicMetrics struct {
		RetweetCount    int `json:"retweet_count"`
		ReplyCount      int `json:"reply_count"`
		LikeCount       int `json:"like_count"`
		QuoteCount      int `json:"quote_count"`
		BookmarkCount   int `json:"bookmark_count"`
		ImpressionCount int `json:"impression_count"`
	}
	TweetReferencedTweets []struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	}
	TweetUser struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
	}
	TweetObject struct {
		ID              string `json:"id"`
		Text            string `json:"text"`
		Lang            string `json:"lang"`
		InReplyToUserID string `json:"in_reply_to_user_id"`
		AuthorID        string `json:"author_id"`
		// Entities
		Entities TweetEntities `json:"entities"`
		// PublicMetrics
		PublicMetrics TweetPublicMetrics `json:"public_metrics"`
		// ReferencedTweets
		ReferencedTweets TweetReferencedTweets `json:"referenced_tweets"`
	}
	TweetResponse struct {
		Data     []TweetObject `json:"data"`
		Includes struct {
			Users  []TweetUser   `json:"users"`
			Tweets []TweetObject `json:"tweets"`
		} `json:"includes"`
		Meta struct {
			ResultCount   int64  `json:"result_count"`
			PreviousToken string `json:"previous_token"`
			NextToken     string `json:"next_token"`
		} `json:"meta"`
	}
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
	url := fmt.Sprintf("https://api.twitter.com/2/lists/%s/tweets", listID)
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

	// fmt.Printf("Response Status: %s\n", resp.Status)
	// fmt.Printf("Response Body: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Errors []struct {
				Message string `json:"message"`
				Code    int    `json:"code"`
			} `json:"errors"`
		}
		if err := json.Unmarshal(body, &errorResponse); err == nil && len(errorResponse.Errors) > 0 {
			return nil, fmt.Errorf("twitter API error: %s (code: %d)", errorResponse.Errors[0].Message, errorResponse.Errors[0].Code)
		}
		return nil, fmt.Errorf("failed to get tweets from list: %s, body: %s", resp.Status, string(body))
	}

	var result TweetResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	return &result, nil
}
