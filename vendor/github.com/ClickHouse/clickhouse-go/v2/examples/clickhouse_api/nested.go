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

	"github.com/ClickHouse/clickhouse-go/v2"
)

func NestedUnFlattened() error {
	conn, err := GetNativeConnection(clickhouse.Settings{
		"flatten_nested": 0,
	}, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	conn.Exec(context.Background(), "DROP TABLE IF EXISTS example")
	err = conn.Exec(ctx, `
		CREATE TABLE example (
			Col1 Nested(Col1_1 String, Col1_2 UInt8),
			Col2 Nested(
			  	Col2_1 UInt8, 
				Col2_2 Nested(
					Col2_2_1 UInt8, 
					Col2_2_2 UInt8
				)
			)
		) Engine Memory
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
			[]map[string]any{
				{
					"Col1_1": strconv.Itoa(int(i)),
					"Col1_2": uint8(i),
				},
				{
					"Col1_1": strconv.Itoa(int(i + 1)),
					"Col1_2": uint8(i + 1),
				},
				{
					"Col1_1": strconv.Itoa(int(i + 2)),
					"Col1_2": uint8(i + 2),
				},
			},
			[]map[string]any{
				{
					"Col2_2": []map[string]any{
						{
							"Col2_2_1": uint8(i),
							"Col2_2_2": uint8(i + 1),
						},
					},
					"Col2_1": uint8(i),
				},
				{
					"Col2_2": []map[string]any{
						{
							"Col2_2_1": uint8(i + 2),
							"Col2_2_2": uint8(i + 3),
						},
					},
					"Col2_1": uint8(i + 1),
				},
			},
		)
		if err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	var (
		col1 []map[string]any
		col2 []map[string]any
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

func NestedFlattened() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	conn.Exec(ctx, "DROP TABLE IF EXISTS example")
	err = conn.Exec(ctx, `
		CREATE TABLE example (
			Col1 Nested(Col1_1 String, Col1_2 UInt8),
			Col2 Nested(
			  	Col2_1 UInt8, 
				Col2_2 Nested(
					Col2_2_1 UInt8, 
					Col2_2_2 UInt8
				)
			)
		) Engine Memory
	`)
	if err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	var i uint8
	for i = 0; i < 10; i++ {
		col1_1_data := []string{strconv.Itoa(int(i)), strconv.Itoa(int(i + 1)), strconv.Itoa(int(i + 2))}
		col1_2_data := []uint8{i, i + 1, i + 2}
		col2_1_data := []uint8{i, i + 1, i + 2}
		col2_2_data := [][][]any{
			{
				{i, i + 1},
			},
			{
				{i + 2, i + 3},
			},
			{
				{i + 4, i + 5},
			},
		}
		err := batch.Append(
			col1_1_data,
			col1_2_data,
			col2_1_data,
			col2_2_data,
		)
		if err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	return nil
}
