package twitter

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/rand"
	"golang.org/x/oauth2"
)

type (
	Client struct {
		cfg         Config
		oauthConfig *oauth2.Config
		rdb         *redis.Client
		httpClient  *http.Client
	}
	Config struct {
		BearerToken  string
		ClientID     string
		ClientSecret string
		CallbackURL  string
	}
	OAuthResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		TokenType    string `json:"token_type"`
		Expiry       string `json:"expiry"`
	}
	UserResponse struct {
		ID              string `json:"id"`
		Username        string `json:"username"`
		Name            string `json:"name"`
		ProfileImageURL string `json:"profile_image_url"`
	}
)

func New(cfg Config, rdb *redis.Client) *Client {
	twitterEndpoint := oauth2.Endpoint{
		AuthURL:  "https://twitter.com/i/oauth2/authorize",
		TokenURL: "https://api.twitter.com/2/oauth2/token",
	}
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		RedirectURL:  cfg.CallbackURL,
		Scopes:       []string{"tweet.read", "tweet.write", "users.read", "offline.access"},
		Endpoint:     twitterEndpoint,
	}

	return &Client{
		cfg:         cfg,
		oauthConfig: oauthConfig,
		rdb:         rdb,
		httpClient:  &http.Client{},
	}
}

func (c *Client) ExchangeTokensWithCode(ctx context.Context, code, state string) (*oauth2.Token, error) {
	key := fmt.Sprintf("user_token:twitter:%s", state)
	codeVerifier, err := c.rdb.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("code_verifier", codeVerifier),
	}

	token, err := c.oauthConfig.Exchange(ctx, code, opts...)
	if err != nil {
		return nil, err
	}

	c.rdb.Del(ctx, key)

	return token, nil
}

func (c *Client) GetAuthURL(ctx context.Context, state string) string {
	codeVerifier := generateCodeVerifier()
	codeChallenge := generateCodeChallenge(codeVerifier)

	key := fmt.Sprintf("user_token:twitter:%s", state)
	c.rdb.Set(ctx, key, codeVerifier, time.Minute*3)

	// with RedirectURL, code_challenge and code_challenge_method
	opts := []oauth2.AuthCodeOption{
		oauth2.SetAuthURLParam("redirect_uri", c.cfg.CallbackURL),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	}

	return c.oauthConfig.AuthCodeURL(state, opts...)
}

func (c *Client) RefreshAccessToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	_token := &oauth2.Token{
		RefreshToken: token.RefreshToken,
	}

	// TokenSource to handle refreshing the token
	tokenSource := c.oauthConfig.TokenSource(ctx, _token)

	// Get a new token
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	return newToken, nil
}

func (c *Client) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserResponse, error) {
	client := c.oauthConfig.Client(ctx, token)

	req, err := http.NewRequest("GET", "https://api.twitter.com/2/users/me?user.fields=profile_image_url", nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info: %s", resp.Status)
	}

	var result struct {
		Data UserResponse `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Data, nil
}

func (c *Client) PostTweet(ctx context.Context, token *oauth2.Token, tweet string) (string, error) {
	client := c.oauthConfig.Client(ctx, token)

	endpoint := "https://api.twitter.com/2/tweets"
	payload := map[string]string{"text": tweet}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// read headers 'x-rate-limit-limit', 'x-rate-limit-remaining', 'x-rate-limit-reset'
	// headers := resp.Header
	// slog.Info("resp.header", "x-rate-limit-limit", headers.Get("x-rate-limit-limit"))
	// slog.Info("resp.header", "x-rate-limit-remaining", headers.Get("x-rate-limit-remaining"))
	// slog.Info("resp.header", "x-rate-limit-reset", headers.Get("x-rate-limit-reset"))

	if resp.StatusCode != http.StatusCreated {
		// read body to get the string
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		// reset the body
		resp.Body = io.NopCloser(bytes.NewBuffer(body))

		return "", fmt.Errorf("failed to post tweet: %s, %s", resp.Status, string(body))
	}

	// decode JSON resp and get the tweet ID
	var result struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Data.ID, nil
}

func generateCodeVerifier() string {
	rand.Seed(uint64(time.Now().UnixNano()))
	length := 43
	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-._~"
	verifier := make([]byte, length)
	for i := range verifier {
		verifier[i] = chars[rand.Intn(len(chars))]
	}
	return string(verifier)
}

func generateCodeChallenge(verifier string) string {
	sha := sha256.New()
	sha.Write([]byte(verifier))
	sum := sha.Sum(nil)
	challenge := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(sum)
	challenge = strings.TrimRight(challenge, "=")
	return challenge
}
