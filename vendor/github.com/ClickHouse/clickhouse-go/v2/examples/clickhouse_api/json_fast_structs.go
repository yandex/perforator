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

type FastProductPricing struct {
	Price    int64  `json:",omitempty"`
	Currency string `json:",omitempty"`
}

type FastProduct struct {
	ID        clickhouse.Dynamic `json:"id"`
	Name      string             `json:"name"`
	Tags      []string           `json:"tags"`
	Pricing   FastProductPricing `json:"pricing"`
	Metadata  map[string]any     `json:"metadata"`
	CreatedAt time.Time          `json:"created_at" chType:"DateTime64(3)"`
}

// SerializeClickHouseJSON implements clickhouse.JSONSerializer for faster struct appending
func (p *FastProduct) SerializeClickHouseJSON() (*clickhouse.JSON, error) {
	obj := clickhouse.NewJSON()
	obj.SetValueAtPath("id", p.ID)
	obj.SetValueAtPath("name", p.Name)
	obj.SetValueAtPath("tags", p.Tags)
	obj.SetValueAtPath("pricing.price", p.Pricing.Price)
	obj.SetValueAtPath("pricing.currency", p.Pricing.Currency)
	obj.SetValueAtPath("metadata.region", p.Metadata["region"])
	obj.SetValueAtPath("metadata.page_count", p.Metadata["page_count"])
	obj.SetValueAtPath("created_at", p.CreatedAt)

	return obj, nil
}

// DeserializeClickHouseJSON implements clickhouse.JSONDeserializer for faster struct scanning
func (p *FastProduct) DeserializeClickHouseJSON(obj *clickhouse.JSON) error {
	p.ID, _ = clickhouse.ExtractJSONPathAs[clickhouse.Dynamic](obj, "id")
	p.Name, _ = clickhouse.ExtractJSONPathAs[string](obj, "name")
	p.Tags, _ = clickhouse.ExtractJSONPathAs[[]string](obj, "tags")
	p.Pricing.Price, _ = clickhouse.ExtractJSONPathAs[int64](obj, "pricing.price")
	p.Pricing.Currency, _ = clickhouse.ExtractJSONPathAs[string](obj, "pricing.currency")
	p.Metadata = make(map[string]any, 2)
	p.Metadata["region"], _ = clickhouse.ExtractJSONPathAs[string](obj, "metadata.region")
	p.Metadata["page_count"], _ = clickhouse.ExtractJSONPathAs[int64](obj, "metadata.page_count")
	p.CreatedAt, _ = clickhouse.ExtractJSONPathAs[time.Time](obj, "created_at")

	return nil
}

func NewExampleFastProduct() *FastProduct {
	return &FastProduct{
		ID:   clickhouse.NewDynamicWithType(uint64(1234), "UInt64"),
		Name: "Book",
		Tags: []string{"library", "fiction"},
		Pricing: FastProductPricing{
			Price:    750,
			Currency: "usd",
		},
		Metadata: map[string]any{
			"region":     "us",
			"page_count": int64(852),
		},
		CreatedAt: time.Now().UTC().Truncate(time.Millisecond),
	}
}

func JSONFastStructExample() error {
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

	insertFastProduct := NewExampleFastProduct()

	if err = batch.Append(insertFastProduct); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	var selectedFastProduct FastProduct

	if err = conn.QueryRow(ctx, "SELECT product FROM go_json_example").Scan(&selectedFastProduct); err != nil {
		return err
	}

	insertFastProductBytes, err := json.Marshal(insertFastProduct)
	if err != nil {
		return err
	}

	selectedFastProductBytes, err := json.Marshal(&selectedFastProduct)
	if err != nil {
		return err
	}

	fmt.Printf("inserted product: %s\n", string(insertFastProductBytes))
	fmt.Printf("selected product: %s\n", string(selectedFastProductBytes))
	fmt.Printf("inserted product matches selected product: %t\n", string(insertFastProductBytes) == string(selectedFastProductBytes))
	return nil
}
