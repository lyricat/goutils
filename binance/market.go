package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

func ListPairs(ctx context.Context, symbols ...string) ([]*Pair, error) {
	params := url.Values{}
	switch len(symbols) {
	case 0:
		// no symbol specified, return all pairs
	case 1:
		params.Set("symbol", symbols[0])
	default:
		// symbols=["BTCUSDT","BNBBTC"]
		raw, _ := json.Marshal(symbols)
		params.Set("symbols", string(raw))
	}

	resp, err := get(ctx, "/api/v3/exchangeInfo", params)
	if err != nil {
		return nil, fmt.Errorf("failed to list pairs: %w", err)
	}

	var data struct {
		Symbols []struct {
			Pair
			Filters []json.RawMessage `json:"filters"`
		} `json:"symbols"`
	}

	if err := json.Unmarshal(resp, &data); err != nil {
		return nil, fmt.Errorf("failed to parse pairs: %w", err)
	}

	var pairs []*Pair
	for _, s := range data.Symbols {
		for _, f := range s.Filters {
			// extract filter type from json.RawMessage
			var filter struct {
				FilterType string `json:"filterType"`
			}

			if err := json.Unmarshal(f, &filter); err != nil {
				return nil, fmt.Errorf("failed to parse filter: %w", err)
			}

			switch filter.FilterType {
			case "PRICE_FILTER":
				_ = json.Unmarshal(f, &s.PriceFilter)
			case "LOT_SIZE":
				_ = json.Unmarshal(f, &s.LotSizeFilter)
			case "MARKET_LOT_SIZE":
				_ = json.Unmarshal(f, &s.MarketLotSizeFilter)
			}
		}
		pairs = append(pairs, &s.Pair)
	}

	return pairs, nil
}

func GetPair(ctx context.Context, symbol string) (*Pair, error) {
	pairs, err := ListPairs(ctx, symbol)
	if err != nil {
		return nil, err
	}

	for _, p := range pairs {
		if p.Symbol == symbol {
			return p, nil
		}
	}

	return nil, fmt.Errorf("pair not found: %s", symbol)
}
