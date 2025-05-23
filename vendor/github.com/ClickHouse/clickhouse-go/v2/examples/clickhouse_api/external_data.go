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

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/ext"
)

func ExternalData() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}

	table1, err := ext.NewTable("external_table_1",
		ext.Column("col1", "UInt8"),
		ext.Column("col2", "String"),
		ext.Column("col3", "DateTime"),
	)
	if err != nil {
		return err
	}

	for i := 0; i < 10; i++ {
		if err = table1.Append(uint8(i), fmt.Sprintf("value_%d", i), time.Now()); err != nil {
			return err
		}
	}

	table2, err := ext.NewTable("external_table_2",
		ext.Column("col1", "UInt8"),
		ext.Column("col2", "String"),
		ext.Column("col3", "DateTime"),
	)

	for i := 0; i < 10; i++ {
		table2.Append(uint8(i), fmt.Sprintf("value_%d", i), time.Now())
	}
	ctx := clickhouse.Context(context.Background(),
		clickhouse.WithExternalTable(table1, table2),
	)
	rows, err := conn.Query(ctx, "SELECT * FROM external_table_1")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			col1 uint8
			col2 string
			col3 time.Time
		)
		rows.Scan(&col1, &col2, &col3)
		fmt.Printf("col1=%d, col2=%s, col3=%v\n", col1, col2, col3)
	}

	var count uint64
	if err := conn.QueryRow(ctx, "SELECT COUNT(*) FROM external_table_1").Scan(&count); err != nil {
		return err
	}
	fmt.Printf("external_table_1: %d\n", count)
	if err := conn.QueryRow(ctx, "SELECT COUNT(*) FROM external_table_2").Scan(&count); err != nil {
		return err
	}
	fmt.Printf("external_table_2: %d\n", count)
	if err := conn.QueryRow(ctx, "SELECT COUNT(*) FROM (SELECT * FROM external_table_1 UNION ALL SELECT * FROM external_table_2)").Scan(&count); err != nil {
		return err
	}
	fmt.Printf("external_table_1 UNION external_table_2: %d\n", count)
	return nil
}
