package twitter

import "time"

type (
	User struct {
		ID              string            `json:"id"`
		Name            string            `json:"name"`
		Username        string            `json:"username"`
		ProfileImageURL string            `json:"profile_image_url"`
		PublicMetrics   UserPublicMetrics `json:"public_metrics"`
	}
	UserPublicMetrics struct {
		FollowersCount uint64 `json:"followers_count"`
		FollowingCount uint64 `json:"following_count"`
		TweetCount     uint64 `json:"tweet_count"`
		ListedCount    uint64 `json:"listed_count"`
	}
)

type (
	ListMemberResponse struct {
		Data []User `json:"data"`
		Meta struct {
			ResultCount   int64  `json:"result_count"`
			PreviousToken string `json:"previous_token"`
			NextToken     string `json:"next_token"`
		} `json:"meta"`
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
		RetweetCount    int64 `json:"retweet_count"`
		ReplyCount      int64 `json:"reply_count"`
		LikeCount       int64 `json:"like_count"`
		QuoteCount      int64 `json:"quote_count"`
		BookmarkCount   int64 `json:"bookmark_count"`
		ImpressionCount int64 `json:"impression_count"`
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

	TweetsResponse struct {
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

	TweetResponse struct {
		Data     TweetObject `json:"data"`
		Includes struct {
			Users  []User        `json:"users"`
			Tweets []TweetObject `json:"tweets"`
		} `json:"includes"`
	}

	SearchIncludes struct {
		Users []User `json:"users"`
	}

	SearchResponse struct {
		Data     []TweetObject  `json:"data"`
		Includes SearchIncludes `json:"includes"`
		Meta     SearchMetadata `json:"meta"`
		Errors   []struct {     // Handle potential API errors
			Message string `json:"message"`
			Detail  string `json:"detail"`
			Title   string `json:"title"`
			Type    string `json:"type"`
		} `json:"errors"`
	}

	SearchMetadata struct {
		NewestID    string `json:"newest_id"`
		OldestID    string `json:"oldest_id"`
		ResultCount int    `json:"result_count"`
		NextToken   string `json:"next_token"`
	}
)

func (t *TweetObject) HasReferencedTweets() bool {
	return len(t.ReferencedTweets) > 0
}

func (t *TweetObject) HasURL() bool {
	return len(t.Entities.Urls) > 0
}

func (t *TweetObject) GetFirstURL() string {
	if t.HasURL() {
		return t.Entities.Urls[0].URL
	}
	return ""
}

func (t *TweetsResponse) GetReferencedTweetByID(id string) *TweetObject {
	for _, tweet := range t.Includes.Tweets {
		if tweet.ID == id {
			return &tweet
		}
	}
	return nil
}

func (t *TweetsResponse) GetUserByID(id string) *User {
	for _, user := range t.Includes.Users {
		if user.ID == id {
			return &user
		}
	}
	return nil
}

func (t *TweetResponse) GetReferencedTweetByID(id string) *TweetObject {
	for _, tweet := range t.Includes.Tweets {
		if tweet.ID == id {
			return &tweet
		}
	}
	return nil
}

func (t *TweetResponse) GetUserByID(id string) *User {
	for _, user := range t.Includes.Users {
		if user.ID == id {
			return &user
		}
	}
	return nil
}
