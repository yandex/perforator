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
	"crypto/tls"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
	"math/rand"
	"time"
)

const TestSet string = "examples_clickhouse_api"

func GetNativeConnection(settings clickhouse.Settings, tlsConfig *tls.Config, compression *clickhouse.Compression) (driver.Conn, error) {
	return clickhouse_tests.GetConnection(TestSet, settings, tlsConfig, compression)
}

func GetNativeTestEnvironment() (clickhouse_tests.ClickHouseTestEnvironment, error) {
	return clickhouse_tests.GetTestEnvironment(TestSet)
}

func GetNativeConnectionWithOptions(settings clickhouse.Settings, tlsConfig *tls.Config, compression *clickhouse.Compression) (driver.Conn, error) {
	return clickhouse_tests.GetConnection(TestSet, settings, tlsConfig, compression)
}

func CheckMinServerVersion(conn driver.Conn, major, minor, patch uint64) bool {
	return clickhouse_tests.CheckMinServerServerVersion(conn, major, minor, patch)
}

var randSeed = time.Now().UnixNano()

func ResetRandSeed() {
	rand.Seed(randSeed)
}
