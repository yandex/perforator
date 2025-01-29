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
	clickHouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests/std"
)

type DatabaseFrame struct {
	name        string
	ColumnNames []string
	rows        *sql.Rows
	columnTypes []*sql.ColumnType
	vars        []any
}

func NewDatabaseFrame(name string, rows *sql.Rows) (DatabaseFrame, error) {
	databaseFrame := DatabaseFrame{}
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return DatabaseFrame{}, err
	}
	databaseFrame.columnTypes = columnTypes
	databaseFrame.name = name
	vars := make([]any, len(columnTypes), len(columnTypes))
	columnNames := make([]string, len(columnTypes), len(columnTypes))
	for i := range columnTypes {
		value := reflect.Zero(columnTypes[i].ScanType()).Interface()
		vars[i] = &value
		columnNames[i] = columnTypes[i].Name()
	}
	databaseFrame.ColumnNames = columnNames
	databaseFrame.vars = vars
	databaseFrame.rows = rows
	return databaseFrame, nil
}

func (f DatabaseFrame) Next() ([]any, bool, error) {
	values := make([]any, len(f.columnTypes), len(f.columnTypes))
	for f.rows.Next() {
		if err := f.rows.Scan(f.vars...); err != nil {
			return nil, false, err
		}
		for i := range f.columnTypes {
			ptr := reflect.ValueOf(f.vars[i])
			values[i] = ptr.Elem().Interface()
		}
		return values, true, nil
	}
	f.rows.Close()
	return nil, false, f.rows.Err()
}

func NewNativeClient(host string, port uint16, username string, password string) (*sql.DB, error) {
	// debug output ?debug=true
	connection, err := clickHouse_tests.GetConnectionFromDSN(fmt.Sprintf("clickhouse://%s:%s@%s:%d/", username, password, host, port))
	if err != nil {
		return nil, err
	}
	if err := connection.Ping(); err != nil {
		return nil, err
	}
	return connection, nil
}

func main() {
	c, err := NewNativeClient("localhost", 9000, "", "")
	if err != nil {
		log.Fatal(err)
	}

	i := 0
	log.Printf("Reading system.%s", "system.query_thread_log")
	rows, err := c.Query("SELECT * FROM system.query_thread_log")
	if err != nil {
		log.Printf("Query failed")
		log.Fatal(err)
	}
	frame, err := NewDatabaseFrame("db_frame", rows)
	if err != nil {
		log.Println("Cant' construct frame")
		log.Fatal(err)
	}
	//iterate to exhaustion

	for {
		_, ok, err := frame.Next()
		if !ok {
			if err != nil {
				log.Println("Failed on termination")
				log.Println(err)
				break
			}
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		i++
	}
	log.Printf("Success with %d rows!!", i)
}
