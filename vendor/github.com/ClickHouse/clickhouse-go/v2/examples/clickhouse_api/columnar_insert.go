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
	"time"
)

func ColumnInsert() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	conn.Exec(ctx, `DROP TABLE IF EXISTS example`)
	if err = conn.Exec(ctx, `
		CREATE TABLE example (
			  Col1 UInt64
			, Col2 String
			, Col3 Array(UInt8)
			, Col4 DateTime
		) ENGINE = Memory
	`); err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(context.Background(), "INSERT INTO example")
	if err != nil {
		return err
	}
	var (
		col1 []uint64
		col2 []string
		col3 [][]uint8
		col4 []time.Time
	)
	for i := 0; i < 1_000; i++ {
		col1 = append(col1, uint64(i))
		col2 = append(col2, "Golang SQL database driver")
		col3 = append(col3, []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9})
		col4 = append(col4, time.Now())
	}
	if err := batch.Column(0).Append(col1); err != nil {
		return err
	}
	if err := batch.Column(1).Append(col2); err != nil {
		return err
	}
	if err := batch.Column(2).Append(col3); err != nil {
		return err
	}
	if err := batch.Column(3).Append(col4); err != nil {
		return err
	}

	// AppendRow is a shortcut for Append(row)
	if err := batch.Column(0).AppendRow(uint64(1_000)); err != nil {
		return err
	}

	if err := batch.Column(1).AppendRow("Golang SQL database driver"); err != nil {
		return err
	}

	if err := batch.Column(2).AppendRow([]uint8{1, 2, 3, 4, 5, 6, 7, 8, 9}); err != nil {
		return err
	}

	if err := batch.Column(3).AppendRow(time.Now()); err != nil {
		return err
	}

	return batch.Send()
}
