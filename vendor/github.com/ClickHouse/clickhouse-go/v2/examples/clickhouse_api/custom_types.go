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

type customStr string

func (s *customStr) Scan(src any) error {
	if t, ok := src.(string); ok {
		*s = customStr(t)
		return nil
	}
	return fmt.Errorf("cannot scan %T into customStr", src)
}

func (s customStr) String() string {
	return string(s)
}

func CustomTypes() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(context.Background(), "DROP TABLE example")
	}()
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS example (
			  Col1 String,
			  Col2 Enum ('hello'   = 1,  'world' = 2)
		) Engine = Memory
	`)
	if err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}

	err = batch.Append(
		customStr("A"), customStr("hello"),
	)
	if err != nil {
		return err
	}

	err = batch.Send()
	if err != nil {
		return err
	}

	var (
		col1 customStr
		col2 customStr
	)

	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&col1, &col2); err != nil {
		return err
	}
	fmt.Printf("col1=%v (T=%T), col2=%v (T=%T)\n", col1, col1, col2, col2)
	return nil
}
