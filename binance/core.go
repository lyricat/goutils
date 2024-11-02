package binance

import (
	"fmt"
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

func (o *Order) PrettyPrint() {
	fmt.Printf("Symbol: %s\n", o.Symbol)
	fmt.Printf("Order ID: %d\n", o.OrderID)
	fmt.Printf("Client Order ID: %s\n", o.ClientOrderID)
	fmt.Printf("Price: %s\n", o.Price)
	fmt.Printf("Original Quantity: %s\n", o.OrigQty)
	fmt.Printf("Executed Quantity: %s\n", o.ExecuteQty)
	fmt.Printf("Cumulative Quote Quantity: %s\n", o.CummulativeQuoteQty)
	fmt.Printf("Status: %s\n", o.Status)
	fmt.Printf("Time in Force: %s\n", o.TimeInForce)
	fmt.Printf("Type: %s\n", o.Type)
	fmt.Printf("Side: %s\n", o.Side)
	fmt.Printf("Stop Price: %s\n", o.StopPrice)
	fmt.Printf("Iceberg Quantity: %s\n", o.IcebergQty)
	fmt.Printf("Time: %s\n", o.Time)
	fmt.Printf("Update Time: %s\n", o.UpdateTime)
	fmt.Printf("Is Working: %v\n", o.IsWorking)
	fmt.Printf("Working Time: %s\n", o.WorkingTime)
	fmt.Printf("Original Quote Order Quantity: %s\n", o.OrigQuoteOrderQty)

}
