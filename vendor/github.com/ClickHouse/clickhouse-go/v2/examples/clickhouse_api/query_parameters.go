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
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
)

func QueryWithParameters() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}

	if !clickhouse_tests.CheckMinServerServerVersion(conn, 22, 8, 0) {
		return nil
	}

	chCtx := clickhouse.Context(context.Background(), clickhouse.WithParameters(clickhouse.Parameters{
		"str":      "hello",
		"array":    "['a', 'b', 'c']",
		"column":   "number",
		"database": "system",
		"table":    "numbers",
	}))

	row := conn.QueryRow(chCtx, "SELECT {column:Identifier} v, {str:String} s, {array:Array(String)} a FROM {database:Identifier}.{table:Identifier} LIMIT 1 OFFSET 100")
	var (
		col1 uint64
		col2 string
		col3 []string
	)
	if err := row.Scan(&col1, &col2, &col3); err != nil {
		return err
	}
	fmt.Printf("row: col1=%d, col2=%s, col3=%s\n", col1, col2, col3)
	return nil
}
