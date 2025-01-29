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
	"net/http"
	_ "net/http/pprof"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickHouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests/std"
)

var conn *sql.DB

func main() {
	go func() {
		http.ListenAndServe("127.0.0.1:9876", nil)
	}()

	var err error
	conn, err = clickHouse_tests.GetConnectionFromDSN("tcp://127.0.0.1:9000?debug=false")
	if err != nil {
		log.Fatal(err)
	}
	conn.SetMaxOpenConns(5)
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(15 * time.Minute)
	if err := conn.Ping(); err != nil {
		if exception, ok := err.(*clickhouse.Exception); ok {
			fmt.Printf("[%d] %s \n%s\n", exception.Code, exception.Message, exception.StackTrace)
		} else {
			fmt.Println(err)
		}
		return
	}
	conn.Exec("DROP TABLE IF EXISTS example")
	_, err = conn.Exec(`
		CREATE EXISTS example (
			country_code FixedString(2),
			os_id        UInt8,
			browser_id   UInt8,
			categories   Array(Int16),
			action_day   Date,
			action_time  DateTime
		) Engine MergeTree() ORDER BY tuple()
	`)
	defer func() {
		conn.Exec("DROP TABLE example")
	}()
	if err != nil {
		log.Fatal(err)
	}

	for range time.Tick(time.Second) {
		log.Println("time", time.Now())
		//go testInsert()
		//go testQuery()
		testQuery()
	}

	if _, err := conn.Exec("DROP TABLE example"); err != nil {
		log.Fatal(err)
	}
}

func testInsert() {
	var (
		tx, _   = conn.Begin()
		stmt, _ = tx.Prepare("INSERT INTO example (country_code, os_id, browser_id, action_day, action_time) VALUES (?, ?, ?, ?, ?)")
	)
	defer stmt.Close()

	for i := 0; i < 100; i++ {
		if _, err := stmt.Exec(
			"RU",
			10+i,
			100+i,
			time.Now(),
			time.Now(),
		); err != nil {
			log.Fatal(err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func testQuery() {
	rows, err := conn.Query("SELECT country_code, os_id, browser_id, categories, action_day, action_time FROM example")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	var (
		country               string
		os, browser           uint8
		categories            []int16
		actionDay, actionTime time.Time
	)
	for rows.Next() {
		if err := rows.Scan(&country, &os, &browser, &categories, &actionDay, &actionTime); err != nil {
			log.Fatal(err)
		}
		//log.Printf("country: %s, os: %d, browser: %d, categories: %v, action_day: %s, action_time: %s", country, os, browser, categories, actionDay, actionTime)
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
}
