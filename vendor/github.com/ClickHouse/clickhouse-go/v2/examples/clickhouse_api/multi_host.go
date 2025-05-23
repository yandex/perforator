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
	"fmt"
	"math/rand"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func MultiHostVersion() error {
	return multiHostVersion(nil)
}

func MultiHostRoundRobinVersion() error {
	connOpenStrategy := clickhouse.ConnOpenRoundRobin
	return multiHostVersion(&connOpenStrategy)
}

func MultiHostRandomVersion() error {
	rand.Seed(85206178671753423)
	defer ResetRandSeed()
	connOpenStrategy := clickhouse.ConnOpenRandom
	return multiHostVersion(&connOpenStrategy)
}

func multiHostVersion(connOpenStrategy *clickhouse.ConnOpenStrategy) error {
	env, err := GetNativeTestEnvironment()
	if err != nil {
		return err
	}
	options := clickhouse.Options{
		Addr: []string{"127.0.0.1:9001", "127.0.0.1:9002", fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
	}
	if connOpenStrategy != nil {
		options.ConnOpenStrategy = *connOpenStrategy
	}
	conn, err := clickhouse.Open(&options)
	if err != nil {
		return err
	}
	v, err := conn.ServerVersion()
	if err != nil {
		return err
	}
	fmt.Println(v.String())
	return nil
}
