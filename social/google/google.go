package google

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

type (
	UserResponse struct {
		ID         string `json:"id"`
		Email      string `json:"email"`
		Name       string `json:"name"`
		GivenName  string `json:"given_name"`
		FamilyName string `json:"family_name"`
		Picture    string `json:"profile"`
	}

	TokenResponse struct {
		Email         string `json:"email"`
		EmailVerified string `json:"email_verified"`
		Sub           string `json:"sub"`
		Scope         string `json:"scope"`
		Exp           string `json:"exp"`
		ExpiresIn     string `json:"expires_in"`
	}

	Config struct {
		CredentialsFile string
	}

	Client struct {
		oauth2Config *oauth2.Config
		rdb          *redis.Client
	}
)

func New(cfg Config, rdb *redis.Client) (*Client, error) {
	credentialsJSON, err := os.ReadFile(cfg.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("reading credentials file: %w", err)
	}

	conf, err := google.ConfigFromJSON(credentialsJSON,
		// The scopes we need:
		"https://www.googleapis.com/auth/userinfo.profile",
		"https://www.googleapis.com/auth/userinfo.email",
	)
	if err != nil {
		return nil, fmt.Errorf("parsing oauth2 config: %w", err)
	}

	return &Client{
		oauth2Config: conf,
		rdb:          rdb,
	}, nil
}

func getCacheKey(state string) string {
	return fmt.Sprintf("user_token:google:%s", state)
}

func (c *Client) GetAuthURL(ctx context.Context, state string) string {
	if c.rdb != nil {
		key := getCacheKey(state)
		c.rdb.Set(ctx, key, state, time.Minute*3)
	}
	return c.oauth2Config.AuthCodeURL(state, oauth2.AccessTypeOffline)
}

func (c *Client) ExchangeTokensWithCode(ctx context.Context, code, state string) (*oauth2.Token, error) {
	var key string
	if c.rdb != nil && state != "" {
		key = getCacheKey(state)
		val, err := c.rdb.Get(ctx, key).Result()
		if err != nil {
			return nil, err
		}

		if val != state {
			return nil, errors.New("invalid state")
		}
	}

	token, err := c.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchanging code for token: %w", err)
	}

	if c.rdb != nil {
		c.rdb.Del(ctx, key)
	}

	return token, nil
}

func (c *Client) RefreshAccessToken(ctx context.Context, token *oauth2.Token) (*oauth2.Token, error) {
	if !token.Valid() && token.RefreshToken == "" {
		return nil, errors.New("cannot refresh without a valid refresh token")
	}

	ts := c.oauth2Config.TokenSource(ctx, token)

	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("refreshing token: %w", err)
	}

	return newToken, nil
}

func (c *Client) GetTokenInfo(ctx context.Context, token *oauth2.Token) (*TokenResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://www.googleapis.com/oauth2/v3/tokeninfo?access_token=%s", token.AccessToken), nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching token info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token info request failed: %s", string(bodyBytes))
	}

	raw := &TokenResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding user info: %w", err)
	}

	if raw.Email == "" {
		return nil, errors.New("email not found")
	}

	if raw.EmailVerified != "true" {
		return nil, errors.New("email not verified")
	}

	return raw, nil
}

func (c *Client) GetUserInfo(ctx context.Context, token *oauth2.Token) (*UserResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v3/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching user info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user info request failed: %s", string(bodyBytes))
	}

	var raw struct {
		ID            string `json:"id"`
		Email         string `json:"email"`
		VerifiedEmail bool   `json:"verified_email"`
		GivenName     string `json:"given_name"`
		FamilyName    string `json:"family_name"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding user info: %w", err)
	}

	if raw.Email == "" {
		return nil, errors.New("email not found")
	}

	if !raw.VerifiedEmail {
		return nil, errors.New("email not verified")
	}

	userInfo := &UserResponse{
		ID:         raw.ID,
		Email:      raw.Email,
		Name:       raw.Name,
		GivenName:  raw.GivenName,
		FamilyName: raw.FamilyName,
		Picture:    raw.Picture,
	}
	return userInfo, nil
}
