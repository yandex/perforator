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
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
	"strconv"
	"time"
)

func QueryRow() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	defer func() {
		conn.Exec("DROP TABLE example")
	}()
	conn.Exec("DROP TABLE IF EXISTS example")
	if _, err := conn.Exec(`
		CREATE TABLE example (
			  Col1 UInt8
			, Col2 String
			, Col3 FixedString(3)
			, Col4 UUID
			, Col5 Map(String, UInt8)
			, Col6 Array(String)
			, Col7 Tuple(String, UInt8, Array(Map(String, String)))
			, Col8 DateTime
		) Engine = Memory
	`); err != nil {
		return err
	}

	scope, err := conn.Begin()
	if err != nil {
		return err
	}
	batch, err := scope.Prepare("INSERT INTO example")
	if err != nil {
		return err
	}
	for i := 0; i < 1000; i++ {
		if _, err := batch.Exec(
			uint8(i),
			"ClickHouse",
			fmt.Sprintf("%03d", uint8(i)),
			uuid.New(),
			map[string]uint8{"key": uint8(i)},
			[]string{strconv.Itoa(i), strconv.Itoa(i + 1), strconv.Itoa(i + 2), strconv.Itoa(i + 3), strconv.Itoa(i + 4), strconv.Itoa(i + 5)},
			[]any{
				strconv.Itoa(i), uint8(i), []map[string]string{
					{"key": strconv.Itoa(i)},
					{"key": strconv.Itoa(i + 1)},
					{"key": strconv.Itoa(i + 2)},
				},
			},
			time.Now(),
		); err != nil {
			return err
		}
	}
	if err := scope.Commit(); err != nil {
		return err
	}
	row := conn.QueryRow("SELECT * FROM example")
	var (
		col1             uint8
		col2, col3, col4 string
		col5             map[string]uint8
		col6             []string
		col7             any
		col8             time.Time
	)
	if err := row.Scan(&col1, &col2, &col3, &col4, &col5, &col6, &col7, &col8); err != nil {
		return err
	}
	fmt.Printf("row: col1=%d, col2=%s, col3=%s, col4=%s, col5=%v, col6=%v, col7=%v, col8=%v\n", col1, col2, col3, col4, col5, col6, col7, col8)
	return nil
}
