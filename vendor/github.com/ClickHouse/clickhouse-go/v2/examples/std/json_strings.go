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

package std

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
)

func JSONStringExample() error {
	ctx := context.Background()

	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}

	if !CheckMinServerVersion(conn, 24, 9, 0) {
		fmt.Print("unsupported clickhouse version for JSON type")
		return nil
	}

	_, err = conn.ExecContext(ctx, "SET allow_experimental_json_type = 1")
	if err != nil {
		return err
	}

	_, err = conn.ExecContext(ctx, "SET output_format_native_write_json_as_string = 1")
	if err != nil {
		return err
	}

	defer func() {
		conn.Exec("DROP TABLE go_json_example")
	}()

	_, err = conn.ExecContext(ctx, "DROP TABLE IF EXISTS go_json_example")
	if err != nil {
		return err
	}

	_, err = conn.ExecContext(ctx, `
		CREATE TABLE go_json_example (
		    product JSON
		) ENGINE = Memory
	`)
	if err != nil {
		return err
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	batch, err := tx.PrepareContext(ctx, "INSERT INTO go_json_example (product)")
	if err != nil {
		return err
	}

	insertProductString := "{\"id\":1234,\"name\":\"Book\",\"tags\":[\"library\",\"fiction\"]," +
		"\"pricing\":{\"price\":750,\"currency\":\"usd\"},\"metadata\":{\"page_count\":852,\"region\":\"us\"}," +
		"\"created_at\":\"2024-12-19T11:20:04.146Z\"}"

	if _, err = batch.ExecContext(ctx, insertProductString); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	var selectedProductString string

	if err = conn.QueryRowContext(ctx, "SELECT product FROM go_json_example").Scan(&selectedProductString); err != nil {
		return err
	}

	fmt.Printf("inserted product string: %s\n", insertProductString)
	fmt.Printf("selected product string: %s\n", selectedProductString)
	fmt.Printf("inserted product string matches selected product string: %t\n", insertProductString == selectedProductString)
	return nil
}
