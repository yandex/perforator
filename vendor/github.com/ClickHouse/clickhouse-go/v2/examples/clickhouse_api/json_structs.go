// Licensed to ClickHouse, Inc. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. ClickHouse, Inc. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package clickhouse_api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type ProductPricing struct {
	Price    int64  `json:",omitempty"`
	Currency string `json:",omitempty"`
}

type Product struct {
	ID        clickhouse.Dynamic     `json:"id"`
	Name      string                 `json:"name"`
	Tags      []string               `json:"tags"`
	Pricing   ProductPricing         `json:"pricing"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at" chType:"DateTime64(3)"`
}

func NewExampleProduct() *Product {
	return &Product{
		ID:   clickhouse.NewDynamicWithType(uint64(1234), "UInt64"),
		Name: "Book",
		Tags: []string{"library", "fiction"},
		Pricing: ProductPricing{
			Price:    750,
			Currency: "usd",
		},
		Metadata: map[string]interface{}{
			"region":     "us",
			"page_count": int64(852),
		},
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
}

func JSONStructExample() error {
	ctx := context.Background()

	conn, err := GetNativeConnection(clickhouse.Settings{
		"allow_experimental_json_type": true,
	}, nil, nil)
	if err != nil {
		return err
	}

	if !CheckMinServerVersion(conn, 24, 9, 0) {
		fmt.Print("unsupported clickhouse version for JSON type")
		return nil
	}

	err = conn.Exec(ctx, "DROP TABLE IF EXISTS go_json_example")
	if err != nil {
		return err
	}

	err = conn.Exec(ctx, `
		CREATE TABLE go_json_example (product JSON) ENGINE=Memory
		`)
	if err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO go_json_example (product)")
	if err != nil {
		return err
	}

	insertProduct := NewExampleProduct()

	if err = batch.Append(insertProduct); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	var selectedProduct Product

	if err = conn.QueryRow(ctx, "SELECT product FROM go_json_example").Scan(&selectedProduct); err != nil {
		return err
	}

	insertProductBytes, err := json.Marshal(insertProduct)
	if err != nil {
		return err
	}

	selectedProductBytes, err := json.Marshal(&selectedProduct)
	if err != nil {
		return err
	}

	fmt.Printf("inserted product: %s\n", string(insertProductBytes))
	fmt.Printf("selected product: %s\n", string(selectedProductBytes))
	fmt.Printf("inserted product matches selected product: %t\n", string(insertProductBytes) == string(selectedProductBytes))
	return nil
}
