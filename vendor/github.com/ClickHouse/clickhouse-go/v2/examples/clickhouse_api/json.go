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

func InsertReadJSON() error {
	conn, err := GetNativeConnection(clickhouse.Settings{
		"allow_experimental_object_type": 1,
	}, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if !clickhouse_tests.CheckMinServerServerVersion(conn, 22, 6, 1) {
		return nil
	}
	if err != nil {
		return nil
	}
	conn.Exec(ctx, "DROP TABLE IF EXISTS example")

	if err = conn.Exec(ctx, `
		CREATE TABLE example (
				Col1 JSON,
				Col2 JSON,
				Col3 JSON
			) 
			Engine Memory
		`); err != nil {
		return err
	}

	type User struct {
		Name     string `json:"name"`
		Age      uint8  `json:"age"`
		Password string `ch:"-"`
	}

	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	// we can insert JSON as either a string, struct or map
	col1Data := `{"name": "Clicky McClickHouse", "age": 40, "password": "password"}`
	col2Data := User{
		Name:     "Clicky McClickHouse Snr",
		Age:      uint8(80),
		Password: "random",
	}
	col3Data := map[string]any{
		"name":     "Clicky McClickHouse Jnr",
		"age":      uint8(10),
		"password": "clicky",
	}
	// both named and unnamed can be added with slices
	if err = batch.Append(col1Data, col2Data, col3Data); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}
	// we can scan JSON into either a map or struct
	var (
		col1 map[string]any
		col2 map[string]any
		col3 User
	)
	// named tuples can be retrieved into a map or slices, unnamed just slices
	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&col1, &col2, &col3); err != nil {
		return err
	}
	fmt.Printf("row: col1=%v, col2=%v, col3=%v\n", col1, col2, col3)

	return nil
}

func ReadComplexJSON() error {
	conn, err := GetNativeConnection(clickhouse.Settings{
		"allow_experimental_object_type": 1,
	}, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()

	if !clickhouse_tests.CheckMinServerServerVersion(conn, 22, 6, 1) {
		return nil
	}
	conn.Exec(ctx, "DROP TABLE IF EXISTS example")

	if err = conn.Exec(ctx, `
		CREATE TABLE example (
				Col1 JSON
			) 
			Engine Memory
		`); err != nil {
		return err
	}

	type Releases struct {
		Version string
	}

	type Repository struct {
		URL      string `json:"url"`
		Releases []Releases
	}

	row := map[string]any{
		"title": "Document JSON support",
		"type":  "Issue",
		"assignee": map[string]any{
			"id":   int16(0),
			"name": "Dale",
			"repositories": []Repository{
				{URL: "https://github.com/ClickHouse/clickhouse-python", Releases: []Releases{{Version: "2.0.0"}, {Version: "2.1.0"}}},
				{URL: "https://github.com/ClickHouse/clickhouse-go"},
			},
			"organizations": []string{},
		},
		"labels": []string{},
		"contributors": []map[string]any{
			{"Id": int16(2244), "Name": "Dale", "orgs": []string{"Support Engineer", "Consulting", "PM", "Integrations"}, "Repositories": []map[string]any{{"url": "https://github.com/ClickHouse/clickhouse-go", "Releases": []map[string]any{{"Version": "2.0.0"}, {"Version": "2.1.0"}}}, {"url": "https://github.com/grafana/clickhouse"}}},
		},
	}
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}

	if err = batch.Append(row); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	var event map[string]any

	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&event); err != nil {
		return err
	}

	fmt.Printf("%v\n", event)
	return nil
}
