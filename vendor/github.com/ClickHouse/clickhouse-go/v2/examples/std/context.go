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
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests/std"
	"github.com/google/uuid"
)

func UseContext() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	if !clickhouse_tests.CheckMinServerVersion(conn, 22, 6, 1) {
		return nil
	}
	// we can use context to pass settings to a specific API call
	ctx := clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
		"async_insert": "1",
	}))
	var settingValue bool
	if err := conn.QueryRowContext(ctx, "SELECT getSetting('async_insert')").Scan(&settingValue); err != nil {
		return fmt.Errorf("failed to get setting value: %v", err)
	}
	if !settingValue {
		return fmt.Errorf("expected setting value to be true, got false")
	}

	// queries can be cancelled using the context
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		cancel()
	}()
	if err = conn.QueryRowContext(ctx, "SELECT sleep(3)").Scan(); err == nil {
		return fmt.Errorf("expected cancel")
	}

	// set a deadline for a query - this will cancel the query after the absolute time is reached. Again terminates the connection only,
	// queries will continue to completion in ClickHouse
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if err := conn.PingContext(ctx); err == nil {
		return fmt.Errorf("expected deadline exceeeded")
	}

	// set a query id to assist tracing queries in logs e.g. see system.query_log
	var one uint8
	ctx = clickhouse.Context(context.Background(), clickhouse.WithQueryID(uuid.NewString()))
	if err = conn.QueryRowContext(ctx, "SELECT 1").Scan(&one); err != nil {
		return err
	}

	conn.ExecContext(context.Background(), "DROP QUOTA IF EXISTS foobar")
	defer func() {
		conn.ExecContext(context.Background(), "DROP QUOTA IF EXISTS foobar")
	}()
	ctx = clickhouse.Context(context.Background(), clickhouse.WithQuotaKey("abcde"))
	// set a quota key - first create the quota
	if _, err = conn.ExecContext(ctx, "CREATE QUOTA IF NOT EXISTS foobar KEYED BY client_key FOR INTERVAL 1 minute MAX queries = 5 TO default"); err != nil {
		return err
	}

	// queries can be cancelled using the context
	ctx, cancel = context.WithCancel(context.Background())
	// we will get some results before cancel
	ctx = clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{
		"max_block_size": "1",
	}))
	rows, err := conn.QueryContext(ctx, "SELECT sleepEachRow(1), number FROM numbers(100);")
	if err != nil {
		return err
	}
	var (
		col1 uint8
		col2 uint8
	)

	for rows.Next() {
		if err := rows.Scan(&col1, &col2); err != nil {
			if col2 > 3 {
				fmt.Println("expected cancel")
				return nil
			}
			return err
		}
		fmt.Printf("row: col2=%d\n", col2)
		if col2 == 3 {
			cancel()
		}
	}
	return nil
}
