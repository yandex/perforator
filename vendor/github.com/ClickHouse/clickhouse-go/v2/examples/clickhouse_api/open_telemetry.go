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
	"go.opentelemetry.io/otel/trace"
)

func OpenTelemetry() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	var count uint64
	rows := conn.QueryRow(clickhouse.Context(context.Background(), clickhouse.WithSpan(
		trace.NewSpanContext(trace.SpanContextConfig{
			SpanID:  trace.SpanID{1, 2, 3, 4, 5},
			TraceID: trace.TraceID{5, 4, 3, 2, 1},
		}),
	)), "SELECT COUNT() FROM (SELECT number FROM system.numbers LIMIT 5)")
	if err := rows.Scan(&count); err != nil {
		return err
	}
	fmt.Printf("count: %d\n", count)
	return nil
}
