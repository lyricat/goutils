package twitter

import "time"

type (
	User struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
	}
)

type (
	ListMemberResponse struct {
		Data []User `json:"data"`
	}
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
		// time
		CreatedAt *time.Time `json:"created_at"`
	}
	TweetResponse struct {
		Data     []TweetObject `json:"data"`
		Includes struct {
			Users  []User        `json:"users"`
			Tweets []TweetObject `json:"tweets"`
		} `json:"includes"`
		Meta struct {
			ResultCount   int64  `json:"result_count"`
			PreviousToken string `json:"previous_token"`
			NextToken     string `json:"next_token"`
		} `json:"meta"`
	}
)
