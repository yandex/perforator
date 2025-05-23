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
	"net"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
	"github.com/google/uuid"
)

func UseContext() error {
	env, err := GetNativeTestEnvironment()
	if err != nil {
		return err
	}
	dialCount := 0
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		DialContext: func(ctx context.Context, addr string) (net.Conn, error) {
			dialCount++
			var d net.Dialer
			return d.DialContext(ctx, "tcp", addr)
		},
	})
	if err != nil {
		return err
	}
	if !clickhouse_tests.CheckMinServerServerVersion(conn, 22, 6, 1) {
		return nil
	}
	// we can use context to pass settings to a specific API call
	ctx := clickhouse.Context(context.Background(), clickhouse.WithSettings(clickhouse.Settings{
		"async_insert": "1",
	}))

	var settingValue bool
	if err := conn.QueryRow(ctx, "SELECT getSetting('async_insert')").Scan(&settingValue); err != nil {
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
	if err = conn.QueryRow(ctx, "SELECT sleep(3)").Scan(); err == nil {
		return fmt.Errorf("expected cancel")
	}

	// set a deadline for a query - this will cancel the query after the absolute time is reached.
	// queries will continue to completion in ClickHouse
	ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	if err := conn.Ping(ctx); err == nil {
		return fmt.Errorf("expected deadline exceeeded")
	}

	// set a query id to assist tracing queries in logs e.g. see system.query_log
	var one uint8
	queryId, _ := uuid.NewUUID()
	ctx = clickhouse.Context(context.Background(), clickhouse.WithQueryID(queryId.String()))
	if err = conn.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil {
		return err
	}

	conn.Exec(context.Background(), "DROP QUOTA IF EXISTS foobar")
	defer func() {
		conn.Exec(context.Background(), "DROP QUOTA IF EXISTS foobar")
	}()
	ctx = clickhouse.Context(context.Background(), clickhouse.WithQuotaKey("abcde"), clickhouse.WithBlockBufferSize(100))
	// set a quota key - first create the quota
	if err = conn.Exec(ctx, "CREATE QUOTA IF NOT EXISTS foobar KEYED BY client_key FOR INTERVAL 1 minute MAX queries = 5 TO default"); err != nil {
		return err
	}

	type Number struct {
		Number uint64 `ch:"number"`
	}
	for i := 1; i <= 5; i++ {
		var result []Number
		if err = conn.Select(ctx, &result, "SELECT number FROM numbers(10)"); err != nil {
			return err
		}
	}
	return nil
}
