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
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Balance struct {
	Type      string  `json:"type,omitempty"`
	Currency  string  `json:"currency,omitempty"`
	Amount    float64 `json:"amount,string,omitempty"`
	Available float64 `json:"available,string,omitempty"`
}

func (c *Client) Balances() ([]*Balance, error) {
	fullURL := fmt.Sprintf("%s/balances", baseURL)
	req, err := http.NewRequest("POST", fullURL, nil)
	if err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"request": "/v1/balances",
		"nonce":   fmt.Sprintf("%v", time.Now().Unix()*1e4),
	}
	blob, _, err := c.doAuthReq(req, payload)
	if err != nil {
		return nil, err
	}
	var balances []*Balance
	if err := json.Unmarshal(blob, &balances); err != nil {
		return nil, err
	}
	return balances, nil
}
