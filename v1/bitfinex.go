// Copyright 2017 orijtech. All Rights Reserved.
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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/orijtech/wsu"
)

type Client struct {
	rt http.RoundTripper
	mu sync.RWMutex
}

func NewClient() (*Client, error) {
	return new(Client), nil
}

type TickerSubscription struct {
	Symbols []string `json:"symbols"`
}

func (c *Client) RealtimeTicker(ts *TickerSubscription) (tkCh chan *Ticker, cancelFunc func() error, err error) {
	conn, err := wsu.NewClientConnection(&wsu.ClientSetup{
		URL: "wss://api.bitfinex.com/ws",
	})
	if err != nil {
		return nil, nil, err
	}

	// 1. Subscribe first
	for _, pair := range ts.Symbols {
		conn.Send(&wsu.Message{
			Frame: []byte(fmt.Sprintf(`{"event": "subscribe", "channel": "ticker", "pair": "%s"}`, pair)),
			WriteAck: func(n int, err error) error {
				return nil
			},
		})
	}

	return makeTickerChan(conn)
}

func makeTickerChan(conn *wsu.Connection) (chan *Ticker, func() error, error) {
	tickerChan := make(chan *Ticker)
	cancelChan, cancelFn := makeCanceler()

	// The cancel go routine
	go func() {
		<-cancelChan
		conn.Close()
	}()

	go func() {
		defer close(tickerChan)

		id := uint64(0)
		doneChan := make(chan bool)
		pairsChannelIDToPair := make(map[float64]string)

		for {
			msg, ok := conn.Receive()
			if !ok {
				return
			}

			id += 1
			go func(m *wsu.Message) {
				defer func() { doneChan <- true }()
				parsed, err := parseIt(msg)
				if err != nil {
					return
				}

				switch pt := parsed.(type) {
				case *Subscription:
					pairsChannelIDToPair[pt.ChannelID] = pt.Pair
				case *Ticker:
					pt.Pair = pairsChannelIDToPair[pt.ChannelID]
					tickerChan <- pt
				default:
					// TODO: Handle this unexpected type
				}
			}(msg)
		}

		for i := uint64(0); i < id; i++ {
			<-doneChan
		}
	}()

	return tickerChan, cancelFn, nil
}

var errAlreadyClosed = errors.New("already closed")

func makeCanceler() (chan bool, func() error) {
	var closeOnce sync.Once
	cancelChan := make(chan bool)
	cancelFn := func() error {
		err := errAlreadyClosed
		closeOnce.Do(func() {
			close(cancelChan)
			err = nil
		})
		return err
	}
	return cancelChan, cancelFn
}

func parseIt(msg *wsu.Message) (interface{}, error) {
	if bytes.HasPrefix(msg.Frame, []byte(`{`)) { // This is a subscription response
		return parseSubscription(msg)
	}
	return parseTicker(msg)
}

func parseTicker(msg *wsu.Message) (*Ticker, error) {
	tk := new(Ticker)
	if err := json.Unmarshal(msg.Frame, tk); err != nil {
		return nil, err
	}
	tk.TimeAt = msg.TimeAt
	return tk, nil
}

func parseSubscription(msg *wsu.Message) (*Subscription, error) {
	subscr := new(Subscription)
	if err := json.Unmarshal(msg.Frame, subscr); err != nil {
		return nil, err
	}
	return subscr, nil
}

type Subscription struct {
	Event     string  `json:"event"`
	Channel   string  `json:"channel"`
	ChannelID float64 `json:"chanId"`
	Pair      string  `json:"pair"`
}

type Ticker struct {
	TimeAt      time.Time `json:"time_at,omitempty"`
	Pair        string    `json:"pair,omitempty"`
	ChannelID   float64   `json:"ch_id,omitempty"`
	Bid         float64   `json:"bid,omitempty"`
	BidSize     float64   `json:"bid_size,omitempty"`
	Ask         float64   `json:"ask,omitempty"`
	AskSize     float64   `json:"ask_size,omitempty"`
	DailyChange float64   `json:"daily_change,omitempty"`
	LastPrice   float64   `json:"last_price,omitempty"`
	Volume      float64   `json:"volume,omitempty"`
	High        float64   `json:"high,omitempty"`
	Low         float64   `json:"low,omitempty"`
	Mid         float64   `json:"mid,omitempty"`

	DailyChangePercentage float64 `json:"daily_change_percent,omitempty"`

	Err error `json:"err,omitempty"`
}

func (tk *Ticker) UnmarshalJSON(b []byte) error {
	// First drill is to parse them as []float64
	var values []float64
	if err := json.Unmarshal(b, &values); err != nil {
		return err
	}
	ptrs := []*float64{
		&tk.ChannelID, &tk.Bid, &tk.BidSize, &tk.Ask,
		&tk.AskSize, &tk.DailyChange, &tk.DailyChangePercentage,
		&tk.LastPrice, &tk.Volume, &tk.High, &tk.Low,
	}
	for i, value := range values {
		if i >= len(ptrs) {
			break
		}
		*(ptrs[i]) = value
	}
	return nil
}

type restTicker struct {
	Mid       float64 `json:"mid,string"`
	Bid       float64 `json:"bid,string"`
	Ask       float64 `json:"ask,string"`
	LastPrice float64 `json:"last_price,string"`
	Low       float64 `json:"low,string"`
	High      float64 `json:"high,string"`
	Volume    float64 `json:"vol,string"`
	Timestamp float64 `json:"timestamp,string"`
}

func (rt *restTicker) toTicker() *Ticker {
	if rt == nil {
		return nil
	}

	s := int64(rt.Timestamp)
	ns := int64((rt.Timestamp - float64(s)) * 1e9)

	return &Ticker{
		Mid:    rt.Mid,
		Bid:    rt.Bid,
		Ask:    rt.Ask,
		Low:    rt.Low,
		High:   rt.High,
		Volume: rt.Volume,

		TimeAt:    time.Unix(s, ns),
		LastPrice: rt.LastPrice,
	}
}

func (c *Client) Ticker(symbol string) (*Ticker, error) {
	symbol = strings.Replace(symbol, "-", "", -1)
	fullURL := "https://api.bitfinex.com/v1/pubticker/" + symbol
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	blob, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	rt := new(restTicker)
	if err := json.Unmarshal(blob, rt); err != nil {
		return nil, err
	}
	return rt.toTicker(), nil
}

func (c *Client) SetHTTPRoundTripper(rt http.RoundTripper) {
	c.mu.Lock()
	c.rt = rt
	c.mu.Unlock()
}

func (c *Client) httpClient() *http.Client {
	c.mu.Lock()
	rt := c.rt
	c.mu.Unlock()
	return &http.Client{Transport: rt}
}
