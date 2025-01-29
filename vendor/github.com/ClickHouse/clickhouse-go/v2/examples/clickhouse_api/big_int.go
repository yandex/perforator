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
	"math/big"
)

func ReadWriteBigInt() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	conn.Exec(ctx, "DROP TABLE IF EXISTS example")

	if err = conn.Exec(ctx, `
		CREATE TABLE example (
			Col1 Int128, 
			Col2 UInt128, 
			Col3 Array(Int128), 
			Col4 Int256, 
			Col5 Array(Int256), 
			Col6 UInt256, 
			Col7 Array(UInt256)
		) Engine Memory`); err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}

	col1Data, _ := new(big.Int).SetString("170141183460469231731687303715884105727", 10)
	col2Data := big.NewInt(128)
	col3Data := []*big.Int{
		big.NewInt(-128),
		big.NewInt(128128),
		big.NewInt(128128128),
	}
	col4Data := big.NewInt(256)
	col5Data := []*big.Int{
		big.NewInt(256),
		big.NewInt(256256),
		big.NewInt(256256256256),
	}
	col6Data := big.NewInt(256)
	col7Data := []*big.Int{
		big.NewInt(256),
		big.NewInt(256256),
		big.NewInt(256256256256),
	}

	if err = batch.Append(col1Data, col2Data, col3Data, col4Data, col5Data, col6Data, col7Data); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	var (
		col1 big.Int
		col2 big.Int
		col3 []*big.Int
		col4 big.Int
		col5 []*big.Int
		col6 big.Int
		col7 []*big.Int
	)

	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&col1, &col2, &col3, &col4, &col5, &col6, &col7); err != nil {
		return err
	}
	fmt.Printf("col1=%v, col2=%v, col3=%v, col4=%v, col5=%v, col6=%v, col7=%v\n", col1, col2, col3, col4, col5, col6, col7)
	return nil
}
