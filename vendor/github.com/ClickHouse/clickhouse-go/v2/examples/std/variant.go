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

func VariantExample() error {
	ctx := context.Background()

	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}

	if !CheckMinServerVersion(conn, 24, 4, 0) {
		fmt.Print("unsupported clickhouse version for Variant type")
		return nil
	}

	_, err = conn.ExecContext(ctx, "SET allow_experimental_variant_type = 1")
	if err != nil {
		return err
	}

	_, err = conn.ExecContext(ctx, "SET allow_suspicious_variant_types = 1")
	if err != nil {
		return err
	}

	defer func() {
		conn.Exec("DROP TABLE go_variant_example")
	}()

	_, err = conn.ExecContext(ctx, "DROP TABLE IF EXISTS go_variant_example")
	if err != nil {
		return err
	}

	_, err = conn.ExecContext(ctx, `
		CREATE TABLE go_variant_example (
		    c Variant(Bool, Int64,  String)
		) ENGINE = Memory
	`)
	if err != nil {
		return err
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	batch, err := tx.PrepareContext(ctx, "INSERT INTO go_variant_example (c)")
	if err != nil {
		return err
	}

	if _, err = batch.ExecContext(ctx, true); err != nil {
		return err
	}

	if _, err = batch.ExecContext(ctx, int64(42)); err != nil {
		return err
	}

	if _, err = batch.ExecContext(ctx, "example"); err != nil {
		return err
	}

	if _, err = batch.ExecContext(ctx, clickhouse.NewVariant("example variant")); err != nil {
		return err
	}

	if _, err = batch.ExecContext(ctx, clickhouse.NewVariantWithType("example variant with specific type", "String")); err != nil {
		return err
	}

	if _, err = batch.ExecContext(ctx, nil); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	// Switch on Go Type

	rows, err := conn.QueryContext(ctx, "SELECT c FROM go_variant_example")
	if err != nil {
		return err
	}

	for i := 0; rows.Next(); i++ {
		var row clickhouse.Variant
		err := rows.Scan(&row)
		if err != nil {
			return fmt.Errorf("failed to scan row index %d: %w", i, err)
		}

		switch row.Any().(type) {
		case bool:
			fmt.Printf("row at index %d is Bool: %v\n", i, row.Any())
		case int64:
			fmt.Printf("row at index %d is Int64: %v\n", i, row.Any())
		case string:
			fmt.Printf("row at index %d is String: %v\n", i, row.Any())
		case nil:
			fmt.Printf("row at index %d is NULL\n", i)
		}
	}

	// Switch on ClickHouse Type

	rows, err = conn.QueryContext(ctx, "SELECT c FROM go_variant_example")
	if err != nil {
		return err
	}

	for i := 0; rows.Next(); i++ {
		var row clickhouse.Variant
		err := rows.Scan(&row)
		if err != nil {
			return fmt.Errorf("failed to scan row index %d: %w", i, err)
		}

		switch row.Type() {
		case "Bool":
			fmt.Printf("row at index %d is bool: %v\n", i, row.Any())
		case "Int64":
			fmt.Printf("row at index %d is int64: %v\n", i, row.Any())
		case "String":
			fmt.Printf("row at index %d is string: %v\n", i, row.Any())
		case "":
			fmt.Printf("row at index %d is nil\n", i)
		}
	}

	return nil
}
