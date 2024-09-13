package twitter

import (
	"context"
	"net/http"

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
