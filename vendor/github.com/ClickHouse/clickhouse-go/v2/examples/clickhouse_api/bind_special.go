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
	"github.com/ClickHouse/clickhouse-go/v2"
	"time"
)

func SpecialBind() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS example (
			Col1 UInt32,
			Col2 String,
			Col3 DateTime64(9),
			Col4 Array(UInt32),
			Col5 UInt32
		) engine=Memory
	`)
	if err != nil {
		return err
	}
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example (Col1, Col2, Col3, Col4, Col5)")
	if err != nil {
		return err
	}
	now := time.Now()
	for i := 0; i < 1000; i++ {
		if err := batch.Append(uint32(i), fmt.Sprintf("value_%d", i), now.Add(time.Duration(i)*time.Millisecond), []uint32{uint32(i), uint32(i) + 1}, uint32(i)+1); err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	var count uint64
	// arrays will be unfolded
	if err = conn.QueryRow(ctx, "SELECT count() FROM example WHERE Col1 IN (?)", []int{100, 200, 300, 400, 500}).Scan(&count); err != nil {
		return err
	}
	fmt.Printf("Array unfolded count: %d\n", count)
	// arrays will be preserved with []
	if err = conn.QueryRow(ctx, "SELECT count() FROM example WHERE Col4 = ?", clickhouse.ArraySet{300, 301}).Scan(&count); err != nil {
		return err
	}
	fmt.Printf("Array count: %d\n", count)
	// Group sets allow us to form ( ) lists
	if err = conn.QueryRow(ctx, "SELECT count() FROM example WHERE Col1 IN ?", clickhouse.GroupSet{[]any{100, 200, 300, 400, 500}}).Scan(&count); err != nil {
		return err
	}
	fmt.Printf("Group count: %d\n", count)
	// More useful when we need nesting
	if err = conn.QueryRow(ctx, "SELECT count() FROM example WHERE (Col1, Col5) IN (?)", []clickhouse.GroupSet{{[]any{100, 101}}, {[]any{200, 201}}}).Scan(&count); err != nil {
		return err
	}
	fmt.Printf("Group count: %d\n", count)
	// Use DateNamed when you need a precision in your time#
	if err = conn.QueryRow(ctx, "SELECT count() FROM example WHERE Col3 >= @col3", clickhouse.DateNamed("col3", now.Add(time.Duration(500)*time.Millisecond), clickhouse.NanoSeconds)).Scan(&count); err != nil {
		return err
	}
	fmt.Printf("NamedDate count: %d\n", count)
	return nil
}
