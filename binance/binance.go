package binance

import (
	"bytes"
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

	"github.com/shopspring/decimal"
)

const (
	baseURL = "https://api.binance.com"
)

type Binance struct {
	APIKey    string
	SecretKey string
	client    *http.Client
}

func New(apiKey, secretKey string) *Binance {
	return &Binance{
		APIKey:    apiKey,
		SecretKey: secretKey,
		client:    &http.Client{},
	}
}

func (c *Binance) QuerySpotOrders(ctx context.Context, symbol string) (string, error) {
	openOrders, err := c.request(ctx, "GET", "/api/v3/openOrders", url.Values{"symbol": {symbol}})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve open orders: %w", err)
	}

	allOrders, err := c.request(ctx, "GET", "/api/v3/allOrders", url.Values{"symbol": {symbol}})
	if err != nil {
		return "", fmt.Errorf("failed to retrieve all orders: %w", err)
	}

	return fmt.Sprintf("Open Orders: %s\nAll Orders (Including Filled): %s", openOrders, allOrders), nil
}

func (c *Binance) GetLatestOrders(ctx context.Context, symbol string, limit int) ([]*Order, error) {
	body, err := c.request(ctx, "GET", "/api/v3/allOrders", url.Values{"symbol": {symbol}, "limit": {strconv.Itoa(limit)}})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve open orders: %w", err)
	}

	var orders []*Order
	if err := json.Unmarshal(body, &orders); err != nil {
		return nil, fmt.Errorf("failed to parse orders: %w", err)
	}

	for _, o := range orders {
		o.Formalize()
	}

	return orders, nil
}

// GetOrderInfoByID retrieves specific order information by order ID for a symbol.
func (c *Binance) GetOrderInfoByID(ctx context.Context, symbol string, orderID int64) (*Order, error) {
	params := url.Values{
		"symbol":  {symbol},
		"orderId": {strconv.FormatInt(orderID, 10)},
	}
	body, err := c.request(ctx, "GET", "/api/v3/order", params)
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

	responseData, err := c.request(ctx, "GET", "/api/v3/myTrades", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}

	var trades []*Trade
	if err := json.Unmarshal([]byte(responseData), &trades); err != nil {
		return nil, fmt.Errorf("failed to parse trades: %w", err)
	}

	for _, t := range trades {
		t.Formalize()
	}

	return trades, nil
}

type PutSpotOrderParams struct {
	NewClientOrderID string
	Symbol           string
	Side             string          // BUY, SELL
	Type             string          // MARKET, LIMIT
	Price            decimal.Decimal // only used for LIMIT
	Quantity         decimal.Decimal
	TimeInForce      string // GTC, IOC, FOK
	NewOrderRespType string // ACK, RESULT, FULL
}

func (c *Binance) PutSpotOrder(ctx context.Context, params PutSpotOrderParams) (*Order, error) {
	values := url.Values{}
	values.Set("symbol", params.Symbol)
	values.Set("side", params.Side)
	values.Set("type", params.Type)
	values.Set("quantity", params.Quantity.String())

	if params.NewClientOrderID != "" {
		values.Set("newClientOrderId", params.NewClientOrderID)
	}

	if params.Type == "LIMIT" {
		values.Set("price", params.Price.String())
	}

	if params.TimeInForce != "" {
		values.Set("timeInForce", params.TimeInForce)
	}

	if params.NewOrderRespType != "" {
		values.Set("newOrderRespType", params.NewOrderRespType)
	}

	resp, err := c.request(ctx, "POST", "/api/v3/order", values)
	if err != nil {
		return nil, fmt.Errorf("failed to put spot order: %w", err)
	}

	var order Order
	if err := json.Unmarshal(resp, &order); err != nil {
		return nil, fmt.Errorf("failed to parse order: %w", err)
	}

	order.Formalize()
	return &order, nil
}

func (c *Binance) CancelSpotOrder(ctx context.Context, symbol string, orderID int64) (*Order, error) {
	params := url.Values{
		"symbol":  {symbol},
		"orderId": {strconv.FormatInt(orderID, 10)},
	}
	resp, err := c.request(ctx, "DELETE", "/api/v3/order", params)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel spot order: %w", err)
	}

	var order Order
	if err := json.Unmarshal(resp, &order); err != nil {
		return nil, fmt.Errorf("failed to parse order: %w", err)
	}

	order.Formalize()
	return &order, nil
}

func (c *Binance) CancelAllSpotOrders(ctx context.Context, symbol string) ([]*Order, error) {
	params := url.Values{
		"symbol": {symbol},
	}

	resp, err := c.request(ctx, "DELETE", "/api/v3/openOrders", params)
	if err != nil {
		return nil, fmt.Errorf("failed to cancel all spot orders: %w", err)
	}

	var orders []*Order
	if err := json.Unmarshal(resp, &orders); err != nil {
		return nil, fmt.Errorf("failed to parse orders: %w", err)
	}

	for _, o := range orders {
		o.Formalize()
	}

	return orders, nil
}

func get(ctx context.Context, endpoint string, params url.Values) ([]byte, error) {
	reqURL := fmt.Sprintf("%s%s", baseURL, endpoint)
	if rawQuery := params.Encode(); rawQuery != "" {
		reqURL += "?" + rawQuery
	}

	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New(string(body))
	}

	return body, nil
}

func (c *Binance) request(ctx context.Context, method, endpoint string, params url.Values) ([]byte, error) {
	header := http.Header{}
	header.Set("X-MBX-APIKEY", c.APIKey)

	query := url.Values{}

	// Add common parameters
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	query.Add("timestamp", timestamp)

	if method == "GET" || method == "DELETE" {
		// merge params
		for k, v := range params {
			query[k] = v
		}
	}

	rawQuery := query.Encode()

	// POST body
	rawBody := ""
	if method == "POST" {
		rawBody = params.Encode()
		header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Generate the signature
	signature := c.sign(rawQuery + rawBody)
	rawQuery += "&signature=" + signature

	// Prepare the request URL
	reqURL := fmt.Sprintf("%s%s?%s", baseURL, endpoint, rawQuery)
	req, err := http.NewRequestWithContext(ctx, method, reqURL, bytes.NewBufferString(rawBody))
	if err != nil {
		return nil, err
	}

	req.Header = header

	// Execute the request
	resp, err := c.client.Do(req)
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
