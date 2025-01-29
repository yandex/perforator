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

import "github.com/ClickHouse/clickhouse-go/v2"

func Exec() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	defer func() {
		conn.Exec("DROP TABLE example")
	}()
	conn.Exec(`DROP TABLE IF EXISTS example`)
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS example (
			Col1 UInt8,
			Col2 String
		) engine=Memory
	`)
	if err != nil {
		return err
	}
	_, err = conn.Exec("INSERT INTO example VALUES (1, 'test-1')")
	return err
}
