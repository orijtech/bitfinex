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
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/orijtech/otils"
	"github.com/orijtech/wsu"
)

type Client struct {
	rt http.RoundTripper
	mu sync.RWMutex

	_apiKey, _apiSecret string
}

func NewClient() (*Client, error) {
	return new(Client), nil
}

const (
	envKeyKey    = "BITFINEX_API_KEY"
	envSecretKey = "BITFINEX_API_SECRET"
)

func NewClientFromEnv() (*Client, error) {
	var errsList []string
	apiKey, errMsg := fromEnv(envKeyKey)
	if errMsg != "" {
		errsList = append(errsList, errMsg)
	}
	apiSecret, errMsg := fromEnv(envSecretKey)
	if errMsg != "" {
		errsList = append(errsList, errMsg)
	}
	if len(errsList) > 0 {
		return nil, errors.New(strings.Join(errsList, "\n"))
	}
	c := &Client{_apiKey: apiKey, _apiSecret: apiSecret}
	return c, nil
}

func fromEnv(key string) (value, errMsg string) {
	if value = os.Getenv(key); value != "" {
		return value, ""
	}
	return "", fmt.Sprintf("%q was not set", key)
}

type Credentials struct {
	Key    string `json:"key"`
	Secret string `json:"secret"`
}

func (c *Client) SetCredentials(ak *Credentials) {
	if ak == nil {
		ak = new(Credentials)
	}
	c.mu.Lock()
	c._apiKey = ak.Key
	c._apiSecret = ak.Secret
	c.mu.Unlock()
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
	tk := new(realTimeTicker)
	if err := json.Unmarshal(msg.Frame, tk); err != nil {
		return nil, err
	}
	tk.TimeAt = msg.TimeAt
	return (*Ticker)(tk), nil
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

type Ticker realTimeTicker

type realTimeTicker struct {
	TimeAt      time.Time `json:"time_at,omitempty"`
	Pair        string    `json:"pair,omitempty"`
	ChannelID   float64   `json:"ch_id,omitempty"`
	Bid         float64   `json:"bid,omitempty"`
	BidSize     float64   `json:"bid_size,omitempty"`
	Ask         float64   `json:"ask,omitempty"`
	AskSize     float64   `json:"ask_size,omitempty"`
	DailyChange float64   `json:"daily_change,omitempty"`
	LastPrice   float64   `json:"last_price,omitempty"`
	High        float64   `json:"high,omitempty"`
	Low         float64   `json:"low,omitempty"`
	Mid         float64   `json:"mid,omitempty"`

	DayVolume   float64 `json:"d_vol,omitempty"`
	WeekVolume  float64 `json:"w_vol,omitempty"`
	MonthVolume float64 `json:"m_vol,omitempty"`

	DailyChangePercentage float64 `json:"daily_change_percent,omitempty"`

	Err error `json:"err,omitempty"`
}

// realTimerTicker messages come in the form of [., ., ., ., .,]
// However, we need Ticker to be unmarshaled as {"...": ...}
// hence the alias of the type as Ticker won't inhert the UnmarshalJSON
func (tk *realTimeTicker) UnmarshalJSON(b []byte) error {
	// First drill is to parse them as []float64
	var values []float64
	if err := json.Unmarshal(b, &values); err != nil {
		return err
	}
	ptrs := []*float64{
		&tk.ChannelID, &tk.Bid, &tk.BidSize, &tk.Ask,
		&tk.AskSize, &tk.DailyChange, &tk.DailyChangePercentage,
		&tk.LastPrice, &tk.DayVolume, &tk.High, &tk.Low,
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
		Mid:       rt.Mid,
		Bid:       rt.Bid,
		Ask:       rt.Ask,
		Low:       rt.Low,
		High:      rt.High,
		DayVolume: rt.Volume,

		TimeAt:    time.Unix(s, ns),
		LastPrice: rt.LastPrice,
	}
}

func bitfinexSymbol(symbol string) string {
	return strings.Replace(symbol, "-", "", -1)
}

const baseURL = "https://api.bitfinex.com/v1"

func (c *Client) Ticker(symbol string) (*Ticker, error) {
	symbol = bitfinexSymbol(symbol)
	fullURL := fmt.Sprintf("%s/pubticker/%s", baseURL, symbol)
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := c.httpClient().Do(req)
	if res.Body != nil {
		defer res.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	if !otils.StatusOK(res.StatusCode) {
		if res.Body != nil {
			if blob, err := ioutil.ReadAll(res.Body); err == nil && len(blob) > 0 {
				return nil, errors.New(string(blob))
			}
		}
		return nil, errors.New(res.Status)
	}
	blob, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	rt := new(restTicker)
	if err := json.Unmarshal(blob, rt); err != nil {
		return nil, err
	}
	tk := rt.toTicker()
	if volumes, _ := c.VolumeInDays(symbol); volumes != nil {
		tk.DayVolume = volumes.Day
		tk.WeekVolume = volumes.Week
		tk.MonthVolume = volumes.Month
	}
	return tk, nil
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

type volume struct {
	Days   float64 `json:"period"`
	Volume float64 `json:"volume,string"`
}

// Volume returns a mapping of days to volume as per
// https://api.bitfinex.com/v1/stats/ETHUSD
// For example
// {
//    1:  100387.9,
//    7:  990387.9,
//    30: 6990387.9,
// }

type Volume struct {
	Day   float64 `json:"day,omitempty"`
	Week  float64 `json:"week,omitempty"`
	Month float64 `json:"month,omitempty"`
}

func (c *Client) VolumeInDays(symbol string) (*Volume, error) {
	symbol = bitfinexSymbol(symbol)
	fullURL := "https://api.bitfinex.com/v1/stats/" + symbol
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
	var volumes []volume
	if err := json.Unmarshal(blob, &volumes); err != nil {
		return nil, err
	}
	periodsMap := make(map[int]float64)
	for _, vol := range volumes {
		periodsMap[int(vol.Days)] = vol.Volume
	}
	vol := &Volume{
		Day:   periodsMap[1],
		Week:  periodsMap[7],
		Month: periodsMap[30],
	}
	return vol, nil
}

func (c *Client) apiKey() string {
	c.mu.Lock()
	key := c._apiKey
	c.mu.Unlock()
	return key
}

func (c *Client) apiSecret() string {
	c.mu.Lock()
	secret := c._apiSecret
	c.mu.Unlock()
	return secret
}

func (c *Client) doAuthReq(req *http.Request, payload interface{}) ([]byte, http.Header, error) {
	blob, err := json.Marshal(payload)
	if err != nil {
		return nil, nil, err
	}
	encodedStr := base64.StdEncoding.EncodeToString(blob)
	sig := hmac.New(sha512.New384, []byte(c.apiSecret()))
	sig.Write([]byte(encodedStr))

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-BFX-APIKEY", c.apiKey())
	req.Header.Add("X-BFX-PAYLOAD", encodedStr)
	req.Header.Add("X-BFX-SIGNATURE", fmt.Sprintf("%x", sig.Sum(nil)))

	res, err := c.httpClient().Do(req)
	if err != nil {
		return nil, nil, err
	}
	if res.Body != nil {
		defer res.Body.Close()
	}
	if !otils.StatusOK(res.StatusCode) {
		if res.Body != nil {
			if blob, err := ioutil.ReadAll(res.Body); err == nil && len(blob) > 0 {
				return nil, res.Header, errors.New(string(blob))
			}
		}
		return nil, res.Header, errors.New(res.Status)
	}
	blob, err = ioutil.ReadAll(res.Body)
	return blob, res.Header, err
}
