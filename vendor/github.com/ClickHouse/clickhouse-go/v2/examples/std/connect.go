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
	"database/sql"
	"fmt"
	"net/url"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func Connect() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
	})
	return conn.Ping()
}

func ConnectDSN() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn, err := sql.Open("clickhouse", fmt.Sprintf("clickhouse://%s:%d?username=%s&password=%s", env.Host, env.Port, env.Username, env.Password))
	if err != nil {
		return err
	}
	return conn.Ping()
}

func ConnectUsingHTTPProxy() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return fmt.Errorf("failed to get test environment: %w", err)
	}

	proxyURL, err := url.Parse("http://proxy.example.com:3128")
	if err != nil {
		return fmt.Errorf("failed to parse proxy URL: %w", err)
	}

	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		HTTPProxyURL: proxyURL,
	})
	return conn.Ping()
}

func ConnectUsingHTTPProxyDSN() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return fmt.Errorf("failed to get test environment: %w", err)
	}

	urlEncodedProxyURL := url.QueryEscape("http://proxy.example.com:3128")

	conn, err := sql.Open("clickhouse", fmt.Sprintf("clickhouse://%s:%d?username=%s&password=%s&http_proxy=%s", env.Host, env.Port, env.Username, env.Password, urlEncodedProxyURL))
	if err != nil {
		return err
	}
	return conn.Ping()
}
