// Copyright 2017 orijtech, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package bitfinex

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/orijtech/otils"
)

type Order struct {
	// Side is required, even in the zero value.
	Side Side `json:"side,omitempty"`

	Type Type `json:"type,omitempty"`

	// Symbol is required, even in the zero value.
	Symbol string `json:"symbol,omitempty"`

	// Price is required, even in the zero value.
	Price float64 `json:"price,string"`

	// Amount is required, even in the zero value.
	Amount float64 `json:"amount,string,omitempty"`

	// Hidden if set, means that the order should be hidden.
	Hidden bool `json:"is_hidden,omitempty"`

	// PostOnly is only relevant for limit orders.
	PostOnly bool `json:"is_postonly,omitempty"`

	// UseAllAvailable if set will post an order
	// that will use all of your available balance.
	UseAllAvailable otils.NumericBool `json:"use_all_available,omitempty"`

	// OCOOrder sets an additional STOP OCO order
	// that will be linked with the current order.
	// OCOOrder is optional but its serialized form
	// must be sent even with the zero value.
	OCOOrder bool `json:"ocoorder"`

	// BuyPriceOCO represents the price of the OCO
	// stop order to place iff OCOOrder is set.
	// BuyPriceOCO is optional but its serialized form
	// must be sent even with the zero value.
	BuyPriceOCO float64 `json:"buy_price_oco"`

	// SellPriceOCO represents the price of the OCO
	// stop order to place if OCOOrder is set.
	// SellPriceOCO is optional but its serialized form
	// must be sent even with the zero value.
	SellPriceOCO float64 `json:"sell_price_oco"`
}

type OrderResponse struct {
	ID       uint64  `json:"id,omitempty"`
	Symbol   string  `json:"symbol,omitempty"`
	Exchange string  `json:"exchange,omitempty"`
	Price    float64 `json:"price,string,omitempty"`

	AverageExecutionPrice float64 `json:"avg_execution_price,string,omitempty"`

	Side            Side    `json:"side,omitempty"`
	Type            string  `json:"type,omitempty"`
	TimestampMs     float64 `json:"timestamp,string,omitempty"`
	Live            bool    `json:"is_live,omitempty"`
	Cancelled       bool    `json:"is_cancelled,omitempty"`
	Hidden          bool    `json:"is_hidden,omitempty"`
	WasForced       bool    `json:"was_forced,omitempty"`
	OriginalAmount  float64 `json:"original_amount,string,omitempty"`
	RemainingAmount float64 `json:"remaining_amount,string,omitempty"`
	ExecutedAmount  float64 `json:"executed_amount,string,omitempty"`
	OrderID         uint64  `json:"order_id,omitempty"`
}

type Type string

const (
	ExchangeLimit        Type = "exchange limit"
	ExchangeMarket       Type = "exchange market"
	Market               Type = "market"
	Limit                Type = "limit"
	Stop                 Type = "stop"
	TrailingStop         Type = "trailing-stop"
	FillOrKill           Type = "fill-or-kill"
	ExchangeTrailingStop Type = "exchange trailing-stop"
	ExchangeFillOrKill   Type = "exchange fill-or-kill"
)

type Side string

const (
	Buy  Side = "buy"
	Sell Side = "sell"
)

var (
	errBlankOrderAmount = errors.New("expecting an order amount")
	errBlankOrderType   = errors.New("expecting an order type")
	errBlankOrderSide   = errors.New("expecting an order side")
	errBlankOrderPrice  = errors.New("expecting an order price")
)

func (o *Order) Validate() error {
	if o == nil || o.Type == "" {
		return errBlankOrderType
	}
	if o.Side == "" {
		return errBlankOrderSide
	}
	if o.Amount <= 0 {
		return errBlankOrderAmount
	}
	if o.Price <= 0 {
		return errBlankOrderPrice
	}
	return nil
}

func (c *Client) Order(order *Order) (*OrderResponse, error) {
	if err := order.Validate(); err != nil {
		return nil, err
	}
	blob, err := json.Marshal(order)
	if err != nil {
		return nil, err
	}
	payload := make(map[string]interface{})
	if err := json.Unmarshal(blob, &payload); err != nil {
		return nil, err
	}

	fullURL := fmt.Sprintf("%s/order/new", baseURL)
	req, err := http.NewRequest("POST", fullURL, nil)
	if err != nil {
		return nil, err
	}

	payload["request"] = "/v1/order/new"
	payload["nonce"] = fmt.Sprintf("%v", time.Now().Unix()*1e4)
	payload["exchange"] = "bitfinex"
	blob, _, err = c.doAuthReq(req, payload)
	if err != nil {
		return nil, err
	}
	resp := new(OrderResponse)
	if err := json.Unmarshal(blob, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Sell is a convenience method to sell a currency instead of invoking Order.
func (c *Client) Sell(amount float64, symbol string, price float64, typ Type) (*OrderResponse, error) {
	return c.Order(&Order{
		Side:   Sell,
		Amount: amount,
		Price:  price,
		Type:   typ,
	})
}

// Buy is a convenience method to buy a currency instead of invoking Order.
func (c *Client) Buy(amount float64, symbol string, price float64, typ Type) (*OrderResponse, error) {
	return c.Order(&Order{
		Side:   Buy,
		Amount: amount,
		Price:  price,
		Type:   typ,
	})
}
