package line

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"gopkg.in/square/go-jose.v2"
)

func (s *Client) GenerateJWTFromJWK(jwkJSON string, kid string) (string, error) {
	// Parse the JWK
	var jwk jose.JSONWebKey
	err := jwk.UnmarshalJSON([]byte(jwkJSON))
	if err != nil {
		return "", err
	}

	// Convert JWK to RSA Private Key
	rsaPrivateKey, ok := jwk.Key.(*rsa.PrivateKey)
	if !ok {
		return "", errors.New("failed to convert JWK to RSA Private Key")
	}

	// Define the token's claims
	claims := jwt.MapClaims{
		"iss":       s.cfg.ChannelID,
		"sub":       s.cfg.ChannelID,
		"aud":       "https://api.line.me/",
		"exp":       time.Now().Add(time.Minute * 29).Unix(), // 29 minutes from now
		"token_exp": 86400,
	}

	// Create a new token with the specified algorithm
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	token.Header["typ"] = "JWT"
	token.Header["alg"] = "RS256"

	// Sign the token with the private key
	return token.SignedString(rsaPrivateKey)
}

func GenerateJWKPair() (string, string, error) {
	var rawkey interface{}
	attrs := map[string]interface{}{
		"alg": "RS256",
		"use": "sig",
	}
	v, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		slog.Error("[goutils.line] failed to generate public key", "error", err)
		return "", "", err
	}
	rawkey = v
	key, err := jwk.FromRaw(rawkey)
	if err != nil {
		slog.Error("[goutils.line] failed to extract public key", "error", err)
		return "", "", err
	}
	for k, v := range attrs {
		if err := key.Set(k, v); err != nil {
			return "", "", err
		}
	}

	keyset := jwk.NewSet()
	keyset.AddKey(key)

	pubks, err := jwk.PublicSetOf(keyset)
	if err != nil {
		return "", "", err
	}

	keybuf, err := json.MarshalIndent(key, "", "  ")
	if err != nil {
		return "", "", err
	}
	pub, _ := pubks.Key(0)
	pubbuf, err := json.MarshalIndent(pub, "", "  ")
	if err != nil {
		return "", "", err
	}

	// encode to base64
	encodedKey := base64.StdEncoding.EncodeToString(keybuf)
	encodedPub := base64.StdEncoding.EncodeToString(pubbuf)

	return encodedPub, encodedKey, err
}

func getChannelAccessToken(jwtToken string) (string, *time.Time, error) {
	// LINE API endpoint for obtaining channel access token
	const tokenEndpoint = "https://api.line.me/oauth2/v2.1/token"

	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	data.Set("client_assertion", jwtToken)

	return getAccessToken(tokenEndpoint, data)
}

func getChannelStatelessAccessToken(jwtToken string) (string, *time.Time, error) {
	// curl -v -X POST https://api.line.me/oauth2/v3/token \
	// -H 'Content-Type: application/x-www-form-urlencoded' \
	// --data-urlencode 'grant_type=client_credentials' \
	// --data-urlencode 'client_assertion_type=urn:ietf:params:oauth:client-assertion-type:jwt-bearer' \
	// --data-urlencode 'client_assertion={JWT assertion}'

	const tokenEndpoint = "https://api.line.me/oauth2/v3/token"
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_assertion_type", "urn:ietf:params:oauth:client-assertion-type:jwt-bearer")
	data.Set("client_assertion", jwtToken)

	return getAccessToken(tokenEndpoint, data)
}

func getAccessToken(url string, data url.Values) (string, *time.Time, error) {
	// Create request
	req, err := http.NewRequest("POST", url, strings.NewReader(data.Encode()))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Create HTTP client and send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, err
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return "", nil, errors.New("failed to get channel access token: " + string(body))
	}

	// Parse response
	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", nil, err
	}

	// Extract access token
	accessToken, ok := result["access_token"].(string)
	if !ok {
		return "", nil, errors.New("access token not found in response")
	}

	expiresInFlt, ok := result["expires_in"].(float64)
	if !ok {
		return "", nil, errors.New("expires_in not found in response")
	}
	expiresIn := int64(expiresInFlt)

	expiredAt := time.Now().Add(time.Duration(expiresIn)*time.Second - 10)

	return accessToken, &expiredAt, nil
}
