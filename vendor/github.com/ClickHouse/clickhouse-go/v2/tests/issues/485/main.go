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
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
)

func example() error {
	var (
		ctx       = context.Background()
		conn, err = clickhouse_tests.GetConnectionWithOptions(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
			//Debug:           true,
			DialTimeout:     time.Second,
			ConnMaxLifetime: time.Hour,
		})
	)
	if err != nil {
		return err
	}
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	err = conn.Exec(ctx, `
 		CREATE TABLE IF NOT EXISTS example (
                           timestamp DateTime64(9) CODEC(Delta, ZSTD(1)),
                           traceID String CODEC(ZSTD(1)),
                           spanID String CODEC(ZSTD(1)),
                           parentSpanID String CODEC(ZSTD(1)),
                           serviceName LowCardinality(String) CODEC(ZSTD(1)),
                           name LowCardinality(String) CODEC(ZSTD(1)),
                           kind Int32 CODEC(ZSTD(1)),
                           durationNano UInt64 CODEC(ZSTD(1)),
                           tags Array(String) CODEC(ZSTD(1)),
                           tagsKeys Array(String) CODEC(ZSTD(1)),
                           tagsValues Array(String) CODEC(ZSTD(1)),
                           statusCode Int64 CODEC(ZSTD(1)),
                           references String CODEC(ZSTD(1)),
                           externalHttpMethod Nullable(String) CODEC(ZSTD(1)),
                           externalHttpUrl Nullable(String) CODEC(ZSTD(1)),
                           component Nullable(String) CODEC(ZSTD(1)),
                           dbSystem Nullable(String) CODEC(ZSTD(1)),
                           dbName Nullable(String) CODEC(ZSTD(1)),
                           dbOperation Nullable(String) CODEC(ZSTD(1)),
                           peerService Nullable(String) CODEC(ZSTD(1)),
                           INDEX idx_traceID traceID TYPE bloom_filter GRANULARITY 4,
                           INDEX idx_service serviceName TYPE bloom_filter GRANULARITY 4,
                           INDEX idx_name name TYPE bloom_filter GRANULARITY 4,
                           INDEX idx_kind kind TYPE minmax GRANULARITY 4,
                           INDEX idx_tagsKeys tagsKeys TYPE bloom_filter(0.01) GRANULARITY 64,
                           INDEX idx_tagsValues tagsValues TYPE bloom_filter(0.01) GRANULARITY 64,
                           INDEX idx_duration durationNano TYPE minmax GRANULARITY 1
                         ) ENGINE MergeTree()
                         PARTITION BY toDate(timestamp)
                         ORDER BY (serviceName, -toUnixTimestamp(timestamp))
 	`)
	if err != nil {
		return err
	}
	for i := 0; i < 10; i++ {
		if err := examplePrep(ctx, conn); err != nil {
			log.Fatal(err)
		}
	}
	return err
}

func examplePrep(ctx context.Context, conn clickhouse.Conn) error {
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	return batch.Send()
}
func main() {
	start := time.Now()
	if err := example(); err != nil {
		log.Fatal(err)
	}
	fmt.Println(time.Since(start))
}
