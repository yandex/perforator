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
	"crypto/tls"
	"database/sql"
	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
	clickhouse_tests_std "github.com/ClickHouse/clickhouse-go/v2/tests/std"
)

const TestSet string = "examples_std_api"

func GetStdDSNConnection(protocol clickhouse.Protocol, secure bool, compress string) (*sql.DB, error) {
	return clickhouse_tests_std.GetDSNConnection(TestSet, protocol, secure, nil)
}

func GetStdOpenDBConnection(protocol clickhouse.Protocol, settings clickhouse.Settings, tlsConfig *tls.Config, compression *clickhouse.Compression) (*sql.DB, error) {
	return clickhouse_tests_std.GetOpenDBConnection(TestSet, protocol, settings, tlsConfig, compression)
}

func GetStdTestEnvironment() (clickhouse_tests.ClickHouseTestEnvironment, error) {
	return clickhouse_tests.GetTestEnvironment(TestSet)
}

func CheckMinServerVersion(conn *sql.DB, major, minor, patch uint64) bool {
	return clickhouse_tests_std.CheckMinServerVersion(conn, major, minor, patch)
}
