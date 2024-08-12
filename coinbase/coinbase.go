package coinbase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shopspring/decimal"
)

const (
	apiBase = "https://api.coinbase.com"
)

type (
	CoinPriceResponse struct {
		Data struct {
			Currency string          `json:"currency"`
			Amount   decimal.Decimal `json:"amount"`
		} `json:"data"`
	}

	ExchangeRates struct {
		Data struct {
			Currency string            `json:"currency"`
			Rates    map[string]string `json:"rates"`
		} `json:"data"`
	}

	CurrencyItems struct {
		Data []struct {
			ID      string          `json:"id"`
			Name    string          `json:"name"`
			MinSize string          `json:"min_size"`
			RateUSD decimal.Decimal `json:"-"`
		} `json:"data"`
	}

	Config struct {
		APIKey    string
		APISecret string
	}

	Coinbase struct {
		cfg Config
	}
)

func New(cfg Config) *Coinbase {
	cb := &Coinbase{cfg: cfg}
	return cb
}

func (cb *Coinbase) GetCryptoPrice(symbol string) (*CoinPriceResponse, error) {

	url := fmt.Sprintf("%s/v2/prices/%s-USD/buy", apiBase, symbol)

	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer response.Body.Close()

	// Read the response body.
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Unmarshal the JSON data.
	var rates CoinPriceResponse
	err = json.Unmarshal(body, &rates)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	return &rates, nil
}

func (cb *Coinbase) GetFiatRatesToUSD() (*CurrencyItems, error) {
	// get fiat currencies
	url := fmt.Sprintf("%s/v2/currencies", apiBase)
	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var items CurrencyItems
	err = json.Unmarshal(body, &items)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	currencyMap := make(map[string]int)
	for _, item := range items.Data {
		currencyMap[item.ID] = 1
	}

	// get exchange rates
	url = fmt.Sprintf("%s/v2/exchange-rates?currency=USD", apiBase)

	response, err = http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making request: %v", err)
	}
	defer response.Body.Close()

	body, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	var rates ExchangeRates
	err = json.Unmarshal(body, &rates)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	fiatRates := make(map[string]string)
	for key, rate := range rates.Data.Rates {
		if _, ok := currencyMap[key]; ok {
			fiatRates[key] = rate
		}
	}

	for i, item := range items.Data {
		if rate, ok := fiatRates[item.ID]; ok {
			items.Data[i].RateUSD, _ = decimal.NewFromString(rate)
		}
	}

	return &items, nil
}
