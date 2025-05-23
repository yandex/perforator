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
	"strconv"
)

func ArrayInsertRead() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	conn.Exec(context.Background(), "DROP TABLE IF EXISTS example")
	err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS example (
			  Col1 Array(String)
			, Col2 Array(Array(Int64))
		) Engine = Memory
	`)
	if err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	var i int64
	for i = 0; i < 10; i++ {
		err := batch.Append(
			[]string{strconv.Itoa(int(i)), strconv.Itoa(int(i + 1)), strconv.Itoa(int(i + 2)), strconv.Itoa(int(i + 3))},
			[][]int64{{i, i + 1}, {i + 2, i + 3}, {i + 4, i + 5}},
		)
		if err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	var (
		col1 []string
		col2 [][]int64
	)
	rows, err := conn.Query(ctx, "SELECT * FROM example")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&col1, &col2); err != nil {
			return err
		}
		fmt.Printf("row: col1=%v, col2=%v\n", col1, col2)
	}
	return rows.Err()
}
