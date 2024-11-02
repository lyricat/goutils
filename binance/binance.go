package binance

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	baseURL = "https://api.binance.com"
)

type Binance struct {
	APIKey    string
	SecretKey string
}

func New(apiKey, secretKey string) *Binance {
	return &Binance{
		APIKey:    apiKey,
		SecretKey: secretKey,
	}
}

func (c *Binance) QuerySpotOrders(ctx context.Context, symbol string) (string, error) {
	openOrders, err := c.request(ctx, "/api/v3/openOrders", url.Values{"symbol": {symbol}})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve open orders: %w", err)
	}

	allOrders, err := c.request(ctx, "/api/v3/allOrders", url.Values{"symbol": {symbol}})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve all orders: %w", err)
	}

	return fmt.Sprintf("Open Orders: %s\nAll Orders (Including Filled): %s", openOrders, allOrders), nil
}

// GetOrderInfoByID retrieves specific order information by order ID for a symbol.
func (c *Binance) GetOrderInfoByID(ctx context.Context, symbol, orderID string) (*Order, error) {
	params := url.Values{
		"symbol":  {symbol},
		"orderId": {orderID},
	}
	body, err := c.request(ctx, "/api/v3/order", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get order info by ID: %w", err)
	}

	ord := &Order{}
	err = json.Unmarshal(body, ord)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	ord.Formalize()

	return ord, nil
}

func (c *Binance) GetTradesByOrderID(ctx context.Context, symbol, orderID string) ([]*Trade, error) {
	params := url.Values{
		"symbol":  {symbol},
		"orderId": {orderID},
	}

	responseData, err := c.request(ctx, "/api/v3/myTrades", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	var trades []*Trade
	if err := json.Unmarshal([]byte(responseData), &trades); err != nil {
		return nil, fmt.Errorf("failed to parse trades: %w", err)
	}

	return trades, nil
}

func (c *Binance) request(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	// Add common parameters
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	params.Add("timestamp", timestamp)

	// Generate the signature
	signature := c.sign(params.Encode())
	params.Add("signature", signature)

	// Prepare the request URL
	reqURL := fmt.Sprintf("%s%s?%s", baseURL, endpoint, params.Encode())
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	// Set the headers
	req.Header.Set("X-MBX-APIKEY", c.APIKey)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	}

	return body, nil
}

func (c *Binance) sign(data string) string {
	h := hmac.New(sha256.New, []byte(c.SecretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}
