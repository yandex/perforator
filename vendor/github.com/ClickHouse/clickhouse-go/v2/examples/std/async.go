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
	"github.com/ClickHouse/clickhouse-go/v2"
)

func AsyncInsert() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	if _, err := conn.Exec(`DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	const ddl = `
		CREATE TABLE example (
			  Col1 UInt64
			, Col2 String
			, Col3 Array(UInt8)
			, Col4 DateTime
		) ENGINE = Memory
		`
	if _, err := conn.Exec(ddl); err != nil {
		return err
	}
	ctx := clickhouse.Context(context.Background(), clickhouse.WithStdAsync(false))
	{
		for i := 0; i < 100; i++ {
			_, err := conn.ExecContext(ctx, `INSERT INTO example VALUES (
				?, ?, ?, now()
			)`, i, "Golang SQL database driver", []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9})
			if err != nil {
				return err
			}
		}
	}
	return nil
}
