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

package bitfinex_test

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/orijtech/bitfinex/v1"
)

const (
	orderRoute       = "/order-make"
	orderCancelRoute = "/order-cancel"
	orderStatusRoute = "/order-status"

	key1    = "^key1$"
	key2    = "     k e72*"
	secret1 = "***$3c5Et."
	secret2 = "$3c***rEt2."
)

var blankOrderResponse = new(bitfinex.OrderResponse)

func TestOrders(t *testing.T) {
	tests := [...]struct {
		key, secret string

		order *bitfinex.Order
		err   string
	}{
		0: {"", "", nil, "order type"},
		1: {key1, secret1, nil, "order type"},
		2: {
			key1, secret1, &bitfinex.Order{
				Type:   bitfinex.ExchangeMarket,
				Side:   bitfinex.Sell,
				Amount: 10,
				Price:  20,
			},
			"",
		},
	}

	for i, tt := range tests {
		client := new(bitfinex.Client)
		client.SetCredentials(&bitfinex.Credentials{Key: tt.key, Secret: tt.secret})
		client.SetHTTPRoundTripper(&testRoundTripper{route: orderRoute})

		res, err := client.Order(tt.order)
		if tt.err != "" {
			if err == nil {
				t.Errorf("#%d: want non-nil error to contain %q", i, tt.err)
			} else if g, w := err.Error(), tt.err; !strings.Contains(g, w) {
				t.Errorf("#%d: got %q want %q", i, g, w)
			}
			continue
		}
		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}
		if res == nil || reflect.DeepEqual(res, blankOrderResponse) {
			t.Errorf("#%d:: want non-blank OrderResposne", i)
		}
	}
}

func TestOrderStatus(t *testing.T) {
	tests := [...]struct {
		key, secret string

		id  interface{}
		err string
	}{
		0: {"", "", nil, "expecting "},
		1: {key1, secret1, nil, "expecting"},
		2: {
			key1, secret1, "448364249", "",
		},
		3: {
			key1, secret1, 448364249, "",
		},
	}

	for i, tt := range tests {
		client := new(bitfinex.Client)
		client.SetCredentials(&bitfinex.Credentials{Key: tt.key, Secret: tt.secret})
		client.SetHTTPRoundTripper(&testRoundTripper{route: orderStatusRoute})

		res, err := client.Status(tt.id)
		if tt.err != "" {
			if err == nil {
				t.Errorf("#%d: want non-nil error to contain %q", i, tt.err)
			} else if g, w := err.Error(), tt.err; !strings.Contains(g, w) {
				t.Errorf("#%d: got %q want %q", i, g, w)
			}
			continue
		}
		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}
		if res == nil || reflect.DeepEqual(res, blankOrderResponse) {
			t.Errorf("#%d:: want non-blank OrderResposne", i)
		}
	}
}

func TestOrderCancel(t *testing.T) {
	tests := [...]struct {
		key, secret string

		id  interface{}
		err string
	}{
		0: {"", "", nil, "expecting "},
		1: {key1, secret1, nil, "expecting"},
		2: {
			key1, secret1, "448364249", "",
		},
		3: {
			key1, secret1, 448364249, "",
		},
	}

	for i, tt := range tests {
		client := new(bitfinex.Client)
		client.SetCredentials(&bitfinex.Credentials{Key: tt.key, Secret: tt.secret})
		client.SetHTTPRoundTripper(&testRoundTripper{route: orderCancelRoute})

		res, err := client.Cancel(tt.id)
		if tt.err != "" {
			if err == nil {
				t.Errorf("#%d: want non-nil error to contain %q", i, tt.err)
			} else if g, w := err.Error(), tt.err; !strings.Contains(g, w) {
				t.Errorf("#%d: got %q want %q", i, g, w)
			}
			continue
		}
		if err != nil {
			t.Errorf("#%d: unexpected error: %v", i, err)
			continue
		}
		if res == nil || reflect.DeepEqual(res, blankOrderResponse) {
			t.Errorf("#%d:: want non-blank OrderResposne", i)
		}
	}
}

type testRoundTripper struct {
	route string
}

var _ http.RoundTripper = (*testRoundTripper)(nil)

func (tr *testRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	switch tr.route {
	case orderRoute:
		return tr.orderRoundTrip(req)
	case orderStatusRoute:
		return tr.orderStatusRoundTrip(req)
	case orderCancelRoute:
		return tr.orderCancelRoundTrip(req)
	default:
		return makeResp("Unimplemented", http.StatusBadRequest, nil), nil
	}
}

func (tr *testRoundTripper) orderStatusRoundTrip(req *http.Request) (*http.Response, error) {
	blob, badRes, err := checkBadAuth(req, "POST")
	if badRes != nil {
		return badRes, nil
	}
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	recv := make(map[string]interface{})
	if err := json.Unmarshal(blob, &recv); err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	orderID, ok := recv["order_id"]
	if !ok {
		return makeResp(`could not find "order_id"`, http.StatusBadRequest, nil), nil
	}
	i64, ok := orderID.(int64)
	if !ok {
		i64 = int64(orderID.(float64))
	}
	fullPath := fmt.Sprintf("./testdata/order-%v.json", i64)
	return respFromFile(fullPath)
}

func (tr *testRoundTripper) orderCancelRoundTrip(req *http.Request) (*http.Response, error) {
	blob, badRes, err := checkBadAuth(req, "POST")
	if badRes != nil {
		return badRes, nil
	}
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	recv := make(map[string]interface{})
	if err := json.Unmarshal(blob, &recv); err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	orderID, ok := recv["id"]
	if !ok {
		return makeResp(`could not find "id"`, http.StatusBadRequest, nil), nil
	}
	i64, ok := orderID.(int64)
	if !ok {
		i64 = int64(orderID.(float64))
	}
	fullPath := fmt.Sprintf("./testdata/order-cancel-%v.json", i64)
	return respFromFile(fullPath)
}

func (tr *testRoundTripper) orderRoundTrip(req *http.Request) (*http.Response, error) {
	blob, badRes, err := checkBadAuth(req, "POST")
	if badRes != nil {
		return badRes, nil
	}
	if err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	order := new(bitfinex.Order)
	if err := json.Unmarshal(blob, order); err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	if err := order.Validate(); err != nil {
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	return respFromFile("./testdata/new-order.json")
}

func checkBadAuth(req *http.Request, wantMethod string) ([]byte, *http.Response, error) {
	if g, w := req.Method, wantMethod; g != w {
		return nil, makeResp(fmt.Sprintf("got method %q want %q", g, w), http.StatusMethodNotAllowed, nil), nil
	}
	body, err := bodyAndAuthSignature(req)
	return body, nil, err
}

var apiSecrets = map[string]string{
	key1: secret1,
	key2: secret2,
}

var errInvalidSignatures = errors.New("invalid/mismatched signatures")

func bodyAndAuthSignature(req *http.Request) ([]byte, error) {
	gotAPIKey := req.Header.Get("X-BFX-APIKEY")
	secretKey, ok := apiSecrets[gotAPIKey]
	if !ok {
		return nil, errors.New("invalid/unknown API Key")
	}
	encodedPayload := req.Header.Get("X-BFX-PAYLOAD")
	if encodedPayload == "" {
		return nil, errors.New(`expected "X-BFX-PAYLOAD"`)
	}
	payload, err := base64.StdEncoding.DecodeString(encodedPayload)
	if err != nil {
		return nil, err
	}
	signature := req.Header.Get("X-BFX-SIGNATURE")

	sig := hmac.New(sha512.New384, []byte(secretKey))
	sig.Write([]byte(encodedPayload))
	if g, w := fmt.Sprintf("%x", sig.Sum(nil)), signature; g != w {
		return payload, errInvalidSignatures
	}
	return payload, nil
}

func respFromFile(p string) (*http.Response, error) {
	f, err := os.Open(p)
	if err != nil {
		if os.IsNotExist(err) {
			return makeResp(err.Error(), http.StatusNotFound, nil), nil
		}
		return makeResp(err.Error(), http.StatusBadRequest, nil), nil
	}
	return makeResp("200 OK", http.StatusOK, f), nil
}

func makeResp(status string, code int, r io.ReadCloser) *http.Response {
	return &http.Response{
		Header:     make(http.Header),
		Body:       r,
		Status:     status,
		StatusCode: code,
	}
}
