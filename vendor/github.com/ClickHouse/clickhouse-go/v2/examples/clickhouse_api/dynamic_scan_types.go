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
	"reflect"
)

func DynamicScan() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	const query = `
	SELECT
		   1     AS Col1
		, 'Text' AS Col2
	`
	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		return err
	}
	var (
		columnTypes = rows.ColumnTypes()
		vars        = make([]any, len(columnTypes))
	)
	for i := range columnTypes {
		vars[i] = reflect.New(columnTypes[i].ScanType()).Interface()
	}
	for rows.Next() {
		if err := rows.Scan(vars...); err != nil {
			return err
		}
		for _, v := range vars {
			switch v := v.(type) {
			case *string:
				fmt.Println(*v)
			case *uint8:
				fmt.Println(*v)
			}
		}
	}
	return nil
}
