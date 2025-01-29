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
	"database/sql"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests/std"
	"time"
)

func BindParameters() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	const ddl = `
	CREATE TABLE example (
		  Col1 UInt8
		, Col2 String
		, Col3 DateTime
	) ENGINE = Memory
	`
	if _, err := conn.Exec(`DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	if _, err := conn.Exec(ddl); err != nil {
		return err
	}
	scope, err := conn.Begin()
	if err != nil {
		return err
	}
	datetime := time.Now()
	{
		scope, err := conn.Begin()
		if err != nil {
			return err
		}
		batch, err := scope.Prepare("INSERT INTO example")
		if err != nil {
			return err
		}
		for i := 0; i < 10; i++ {
			if _, err := batch.Exec(uint8(i), "ClickHouse Inc.", datetime.Add(time.Duration(i)*time.Hour)); err != nil {
				return err
			}
		}
		if err := scope.Commit(); err != nil {
			return err
		}
	}
	if err := scope.Commit(); err != nil {
		return err
	}

	var result struct {
		Col1 uint8
		Col2 string
		Col3 time.Time
	}

	// numeric bind
	if err := conn.QueryRow(`SELECT * FROM example WHERE Col1 = $1 AND Col3 = $2`, 2, datetime.Add(time.Duration(2)*time.Hour)).Scan(
		&result.Col1,
		&result.Col2,
		&result.Col3,
	); err != nil {
		return err
	}
	fmt.Println(clickhouse_tests.ToJson(result))

	// named bind
	if err := conn.QueryRow(`SELECT * FROM example WHERE Col1 = @Col1 AND Col3 = @Col2`,
		sql.Named("Col1", 4),
		sql.Named("Col2", datetime.Add(time.Duration(4)*time.Hour)),
	).Scan(
		&result.Col1,
		&result.Col2,
		&result.Col3,
	); err != nil {
		return err
	}
	fmt.Println(clickhouse_tests.ToJson(result))

	// positional bind
	if err := conn.QueryRow(`SELECT * FROM example WHERE Col1 = ? AND Col3 = ?`,
		6,
		datetime.Add(time.Duration(6)*time.Hour),
	).Scan(
		&result.Col1,
		&result.Col2,
		&result.Col3,
	); err != nil {
		return err
	}
	fmt.Println(clickhouse_tests.ToJson(result))

	return nil
}
