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
	"encoding/json"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/proto"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func GetStdTestEnvironment() (clickhouse_tests.ClickHouseTestEnvironment, error) {
	return clickhouse_tests.GetTestEnvironment("std")
}

func CheckMinServerVersion(conn *sql.DB, major, minor, patch uint64) bool {
	var res string
	if err := conn.QueryRow("SELECT version()").Scan(&res); err != nil {
		panic(err)
	}
	var version proto.Version
	for i, v := range strings.Split(res, ".") {
		switch i {
		case 0:
			version.Major, _ = strconv.ParseUint(v, 10, 64)
		case 1:
			version.Minor, _ = strconv.ParseUint(v, 10, 64)
		case 2:
			version.Patch, _ = strconv.ParseUint(v, 10, 64)
		}
	}
	return proto.CheckMinVersion(proto.Version{
		Major: major,
		Minor: minor,
		Patch: patch,
	}, version)
}

func GetDSNConnection(environment string, protocol clickhouse.Protocol, secure bool, opts url.Values) (*sql.DB, error) {
	env, err := clickhouse_tests.GetTestEnvironment(environment)
	if err != nil {
		return nil, err
	}
	insertQuorum := clickhouse_tests.GetEnv("CLICKHOUSE_QUORUM_INSERT", "1")

	scheme := "clickhouse"
	port := env.Port

	query := opts
	if query == nil {
		query = make(url.Values)
	}

	query.Set("insert_quorum", insertQuorum)
	query.Set("insert_quorum_parallel", "0")
	query.Set("select_sequential_consistency", "1")

	if proto.CheckMinVersion(proto.Version{
		Major: 22,
		Minor: 8,
		Patch: 0,
	}, env.Version) {
		query.Set("database_replicated_enforce_synchronous_settings", "1")
	}

	if protocol == clickhouse.HTTP {
		query.Set("wait_end_of_query", "1")

		if secure {
			scheme = "https"
			port = env.HttpsPort
		} else {
			scheme = "http"
			port = env.HttpPort
		}
	} else {
		if secure {
			port = env.SslPort
		}
	}

	if secure {
		query.Set("secure", "true")
	}

	dsn := url.URL{
		Scheme:   scheme,
		User:     url.UserPassword(env.Username, env.Password),
		Host:     fmt.Sprintf("%s:%d", env.Host, port),
		Path:     env.Database,
		RawQuery: query.Encode(),
	}

	return sql.Open("clickhouse", dsn.String())
}

func GetConnectionFromDSN(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("clickhouse", dsn)
	if err != nil {
		return conn, err
	}
	if CheckMinServerVersion(conn, 22, 8, 0) {
		dsn = fmt.Sprintf("%s&database_replicated_enforce_synchronous_settings=1", dsn)
	}
	insertQuorum := clickhouse_tests.GetEnv("CLICKHOUSE_QUORUM_INSERT", "1")
	dsn = fmt.Sprintf("%s&insert_quorum=%s&insert_quorum_parallel=0&select_sequential_consistency=1", dsn, insertQuorum)
	if strings.HasPrefix(dsn, "http") {
		dsn = fmt.Sprintf("%s&wait_end_of_query=1", dsn)
	}
	return sql.Open("clickhouse", dsn)
}

func GetConnectionWithOptions(options *clickhouse.Options) *sql.DB {
	if options.Settings == nil {
		options.Settings = clickhouse.Settings{}
	}
	conn := clickhouse.OpenDB(options)
	if CheckMinServerVersion(conn, 22, 8, 0) {
		options.Settings["database_replicated_enforce_synchronous_settings"] = "1"
	}
	var err error
	options.Settings["insert_quorum"], err = strconv.Atoi(clickhouse_tests.GetEnv("CLICKHOUSE_QUORUM_INSERT", "1"))
	options.Settings["insert_quorum_parallel"] = 0
	options.Settings["select_sequential_consistency"] = 1
	if err != nil {
		return nil
	}
	return clickhouse.OpenDB(options)
}

func GetOpenDBConnection(environment string, protocol clickhouse.Protocol, settings clickhouse.Settings, tlsConfig *tls.Config, compression *clickhouse.Compression) (*sql.DB, error) {
	env, err := clickhouse_tests.GetTestEnvironment(environment)
	if err != nil {
		return nil, err
	}
	var port int
	switch protocol {
	case clickhouse.HTTP:
		port = env.HttpPort
		if tlsConfig != nil {
			port = env.HttpsPort
		}
	case clickhouse.Native:
		port = env.Port
		if tlsConfig != nil {
			port = env.SslPort
		}
	}
	if settings == nil {
		settings = clickhouse.Settings{}
	}
	if protocol == clickhouse.HTTP {
		settings["wait_end_of_query"] = 1
	}
	settings["insert_quorum"], err = strconv.Atoi(clickhouse_tests.GetEnv("CLICKHOUSE_QUORUM_INSERT", "1"))
	settings["insert_quorum_parallel"] = 0
	settings["select_sequential_consistency"] = 1
	if proto.CheckMinVersion(proto.Version{
		Major: 22,
		Minor: 8,
		Patch: 0,
	}, env.Version) {
		settings["database_replicated_enforce_synchronous_settings"] = "1"
	}
	if err != nil {
		return nil, err
	}
	return clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		Settings:    settings,
		DialTimeout: 5 * time.Second,
		Compression: compression,
		TLS:         tlsConfig,
		Protocol:    protocol,
	}), nil
}

func ToJson(obj any) string {
	bytes, err := json.Marshal(obj)
	if err != nil {
		return "unable to marshal"
	}
	return string(bytes)
}
