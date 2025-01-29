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
	"fmt"
)

func TupleInsertRead() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn.Exec(ctx, "DROP TABLE IF EXISTS example")

	if err = conn.Exec(ctx, `
		CREATE TABLE example (
				Col1 Tuple(name String, age UInt8),
				Col2 Tuple(String, UInt8),
				Col3 Tuple(name String, id String)
			) 
			Engine Memory
		`); err != nil {
		return err
	}

	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	// both named and unnamed can be added with slices. Note we can use strongly typed lists and maps if all elements are the same type
	if err = batch.Append([]any{"Clicky McClickHouse", uint8(42)}, []any{"Clicky McClickHouse Snr", uint8(78)}, []string{"Dale", "521211"}); err != nil {
		return err
	}
	if err = batch.Append(map[string]any{"name": "Clicky McClickHouse Jnr", "age": uint8(20)}, []any{"Baby Clicky McClickHouse", uint8(1)}, map[string]string{"name": "Geoff", "id": "12123"}); err != nil {
		return err
	}
	if err = batch.Send(); err != nil {
		return err
	}
	var (
		col1 map[string]any
		col2 []any
		col3 map[string]string
	)
	// named tuples can be retrieved into a map or slices, unnamed just slices
	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&col1, &col2, &col3); err != nil {
		return err
	}
	fmt.Printf("row: col1=%v, col2=%v, col3=%v\n", col1, col2, col3)

	return nil
}
