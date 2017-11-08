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

package bitfinex_test

import (
	"log"

	"github.com/orijtech/bitfinex"
)

func Example_client_Subscribe() {
	client, err := bitfinex.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	tkChan, cancelFn, err := client.RealtimeTicker(&bitfinex.TickerSubscription{
		Symbols: []string{
			"BTCUSD",
			"LTCUSD",
			"LTCBTC",
			"ETHUSD",
			"ETHBTC",
			"ETCBTC",
			"ETCUSD",
			"BCHUSD",
			"BCHBTC",
			"BCHETH",
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer cancelFn()

	for ticker := range tkChan {
		log.Printf("ticker: %+v\n", ticker)
	}
}
