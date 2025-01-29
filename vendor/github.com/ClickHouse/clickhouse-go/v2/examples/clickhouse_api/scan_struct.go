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
	"time"
)

func ScanStruct() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}

	const ddl = `
	CREATE TABLE example (
		  Col1 Int64
		, Col2 FixedString(2)
		, Col3 Map(String, Int64)
		, Col4 Array(Int64)
		, Col5 DateTime64(3)
	) Engine Memory
	`
	defer func() {
		conn.Exec(context.Background(), "DROP TABLE example")
	}()
	if err := conn.Exec(context.Background(), "DROP TABLE IF EXISTS example"); err != nil {
		return err
	}
	if err := conn.Exec(context.Background(), ddl); err != nil {
		return err
	}
	batch, err := conn.PrepareBatch(context.Background(), "INSERT INTO example")
	if err != nil {
		return err
	}
	for i := int64(0); i < 1000; i++ {
		err := batch.Append(
			i%10,
			"CH",
			map[string]int64{
				"key": i,
			},
			[]int64{i, i + 1, i + 2},
			time.Now(),
		)
		if err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}

	var result struct {
		Col1  int64
		Count uint64 `ch:"count"`
	}
	if err := conn.QueryRow(context.Background(), "SELECT Col1, COUNT() AS count FROM example WHERE Col1 = 5 GROUP BY Col1").ScanStruct(&result); err != nil {
		return err
	}
	fmt.Println("result", result)
	return nil
}
