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

	"github.com/orijtech/bitfinex/v1"
)

func Example_client_RealtimeTicker() {
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

func Example_client_Ticker() {
	client, err := bitfinex.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	tk, err := client.Ticker("BTC-USD")
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("current ticker: %+v\n", tk)
}

func Example_client_Balances() {
	client, err := bitfinex.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	balances, err := client.Balances()
	if err != nil {
		log.Fatal(err)
	}
	for i, balance := range balances {
		log.Printf("#%d: %+v\n", i, balance)
	}
}

func Example_client_Buy() {
	client, err := bitfinex.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	// Buy 3/4 BTC at $7500USD
	res, err := client.Buy(0.75, "BTCUSD", 7500, bitfinex.Limit)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Response: %+v\n", res)
}

func Example_client_Sell() {
	client, err := bitfinex.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	// Sell 107.3 ETH at $400USD
	res, err := client.Buy(107.3, "ETHUSD", 7500, bitfinex.Limit)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Response: %+v\n", res)
}

func Example_client_Order() {
	client, err := bitfinex.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	// Sell off all the LTC that we've got.
	res, err := client.Order(&bitfinex.Order{
		Side:            bitfinex.Sell,
		Symbol:          "ETHBTC",
		Price:           400,
		Type:            bitfinex.FillOrKill,
		UseAllAvailable: true,
		Hidden:          true,
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Response: %+v\n", res)
}

func Example_client_Status() {
	client, err := bitfinex.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.Status(446915287)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Response: %+v\n", res)
}

func Example_client_Cancel() {
	client, err := bitfinex.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	res, err := client.Cancel(446915287)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Response: %+v\n", res)
}
