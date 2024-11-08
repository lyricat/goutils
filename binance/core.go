package binance

import (
	"fmt"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type (
	Order struct {
		Symbol              string          `json:"symbol"`
		OrderID             int64           `json:"orderId"`
		OrderListID         int64           `json:"orderListId"`
		ClientOrderID       string          `json:"clientOrderId"`
		Price               decimal.Decimal `json:"price"`
		OrigQty             decimal.Decimal `json:"origQty"`
		ExecuteQty          decimal.Decimal `json:"executedQty"`
		CummulativeQuoteQty decimal.Decimal `json:"cummulativeQuoteQty"`
		Status              string          `json:"status"`
		TimeInForce         string          `json:"timeInForce"`
		Type                string          `json:"type"`
		Side                string          `json:"side"`
		StopPrice           decimal.Decimal `json:"stopPrice"`
		IcebergQty          decimal.Decimal `json:"icebergQty"`
		UnixTime            uint64          `json:"time"`
		UnixUpdateTime      uint64          `json:"updateTime"`
		IsWorking           bool            `json:"isWorking"`
		UnixWorkingTime     uint64          `json:"workingTime"`
		OrigQuoteOrderQty   decimal.Decimal `json:"origQuoteOrderQty"`

		Time        time.Time `json:"-"`
		UpdateTime  time.Time `json:"-"`
		WorkingTime time.Time `json:"-"`
	}

	Trade struct {
		ID              int64           `json:"id"`
		Symbol          string          `json:"symbol"`
		OrderID         int64           `json:"orderId"`
		Price           decimal.Decimal `json:"price"`
		Qty             decimal.Decimal `json:"qty"`
		Commission      decimal.Decimal `json:"commission"`
		CommissionAsset string          `json:"commissionAsset"`
		UnixTime        uint64          `json:"time"`
		IsBuyer         bool            `json:"isBuyer"`
		IsMaker         bool            `json:"isMaker"`
		IsBestMatch     bool            `json:"isBestMatch"`

		Time time.Time `json:"-"`
	}

	Fill struct {
		Price           string `json:"price"`
		Qty             string `json:"qty"`
		Commission      string `json:"commission"`
		CommissionAsset string `json:"commissionAsset"`
	}
)

type (
	PriceFilter struct {
		FilterType string          `json:"filterType"`
		MinPrice   decimal.Decimal `json:"minPrice"`
		MaxPrice   decimal.Decimal `json:"maxPrice"`
		TickSize   decimal.Decimal `json:"tickSize"`
	}

	LotSizeFilter struct {
		FilterType string          `json:"filterType"`
		MinQty     decimal.Decimal `json:"minQty"`
		MaxQty     decimal.Decimal `json:"maxQty"`
		StepSize   decimal.Decimal `json:"stepSize"`
	}

	MarketLotSizeFilter struct {
		FilterType string          `json:"filterType"`
		MinQty     decimal.Decimal `json:"minQty"`
		MaxQty     decimal.Decimal `json:"maxQty"`
		StepSize   decimal.Decimal `json:"stepSize"`
	}

	Pair struct {
		Symbol              string              `json:"symbol"`
		BaseAsset           string              `json:"baseAsset"`
		BaseAssetPrecision  int                 `json:"baseAssetPrecision"`
		QuoteAsset          string              `json:"quoteAsset"`
		QuoteAssetPrecision int                 `json:"quoteAssetPrecision"`
		QuotePrecision      int                 `json:"quotePrecision"`
		PriceFilter         PriceFilter         `json:"priceFilter"`
		LotSizeFilter       LotSizeFilter       `json:"lotSizeFilter"`
		MarketLotSizeFilter MarketLotSizeFilter `json:"marketLotSizeFilter"`
	}
)

func (o *Order) Formalize() {
	// convert timestamps into time.Time
	if o.UnixTime > 0 {
		o.Time = time.Unix(0, int64(o.UnixTime)*int64(time.Millisecond))
	}
	if o.UnixUpdateTime > 0 {
		o.UpdateTime = time.Unix(0, int64(o.UnixUpdateTime)*int64(time.Millisecond))
	}
	if o.UnixWorkingTime > 0 {
		o.WorkingTime = time.Unix(0, int64(o.UnixWorkingTime)*int64(time.Millisecond))
	}
}

func (t *Trade) Formalize() {
	if t.UnixTime > 0 {
		t.Time = time.Unix(0, int64(t.UnixTime)*int64(time.Millisecond))
	}
}

func (o *Order) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Symbol: %s\n", o.Symbol))
	b.WriteString(fmt.Sprintf("Order ID: %d\n", o.OrderID))
	b.WriteString(fmt.Sprintf("Client Order ID: %s\n", o.ClientOrderID))
	b.WriteString(fmt.Sprintf("Price: %s\n", o.Price))
	b.WriteString(fmt.Sprintf("Original Quantity: %s\n", o.OrigQty))
	b.WriteString(fmt.Sprintf("Executed Quantity: %s\n", o.ExecuteQty))
	b.WriteString(fmt.Sprintf("Cumulative Quote Quantity: %s\n", o.CummulativeQuoteQty))
	b.WriteString(fmt.Sprintf("Status: %s\n", o.Status))
	b.WriteString(fmt.Sprintf("Time in Force: %s\n", o.TimeInForce))
	b.WriteString(fmt.Sprintf("Type: %s\n", o.Type))
	b.WriteString(fmt.Sprintf("Side: %s\n", o.Side))
	b.WriteString(fmt.Sprintf("Stop Price: %s\n", o.StopPrice))
	b.WriteString(fmt.Sprintf("Iceberg Quantity: %s\n", o.IcebergQty))
	b.WriteString(fmt.Sprintf("Time: %s\n", o.Time))
	b.WriteString(fmt.Sprintf("Update Time: %s\n", o.UpdateTime))
	b.WriteString(fmt.Sprintf("Is Working: %v\n", o.IsWorking))
	b.WriteString(fmt.Sprintf("Working Time: %s\n", o.WorkingTime))
	b.WriteString(fmt.Sprintf("Original Quote Order Quantity: %s\n", o.OrigQuoteOrderQty))
	return b.String()
}
