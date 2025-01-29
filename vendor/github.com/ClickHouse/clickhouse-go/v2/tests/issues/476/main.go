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

package main

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

func main() {
	c, err := sql.Open("clickhouse", fmt.Sprintf("clickhouse://%s:%s@%s:%d/", "", "", "localhost", 9000))
	if err != nil {
		log.Printf("Can't create connection to db")
		log.Fatal(err)
	}
	if err := c.Ping(); err != nil {
		log.Printf("Can't connect to db")
		log.Fatal(err)
	}

	log.Println("Reading system.query_log")
	rows, err := c.Query("SELECT * FROM system.query_log")
	if err != nil {
		log.Printf("Query failed")
		log.Fatal(err)
	}
	//iterate to exhaustion
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		log.Printf("Can't get column types")
		log.Fatal(err)
	}
	vars := make([]any, len(columnTypes), len(columnTypes))
	for i := range columnTypes {
		value := reflect.Zero(columnTypes[i].ScanType()).Interface()
		vars[i] = &value
	}
	i := 0
	for rows.Next() {
		if err := rows.Scan(vars...); err != nil {
			log.Fatal(err)
		}
		i++
	}
	log.Println(rows.Err())
	log.Printf("Success with %d rows!!", i)
}
