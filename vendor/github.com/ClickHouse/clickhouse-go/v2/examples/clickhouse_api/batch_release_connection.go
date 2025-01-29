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
	"errors"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

func BatchWithReleaseConnection() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS example (
			Col1 UInt64,
			Col2 String
		) engine=Memory
	`)

	batch, err := New(ctx, conn, "INSERT INTO example")
	if err != nil {
		return err
	}

	if err = batch.Append(1, "test-1"); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	if err = batch.Append(2, "test-2"); err != nil {
		return err
	}

	if err = batch.Send(); err != nil {
		return err
	}

	var count uint64
	if err = conn.QueryRow(context.Background(), `SELECT COUNT(*) FROM example`).Scan(&count); err != nil {
		return err
	}

	if count != uint64(2) {
		return errors.New("count must be 2")
	}

	return nil
}

type YourBatch struct {
	ctx context.Context

	insertStatement string

	conn  driver.Conn
	batch driver.Batch
}

func New(ctx context.Context, conn driver.Conn, insertStatement string) (*YourBatch, error) {
	batch, err := conn.PrepareBatch(ctx, insertStatement, driver.WithReleaseConnection())
	if err != nil {
		return nil, err
	}

	return &YourBatch{
		ctx:             ctx,
		insertStatement: insertStatement,
		conn:            conn,
		batch:           batch,
	}, nil
}

func (b *YourBatch) Append(col1 uint64, col2 string) error {
	return b.batch.Append(
		col1,
		col2,
	)
}

func (b *YourBatch) Send() error {
	if err := b.batch.Send(); err != nil {
		return err
	}

	return b.reset()
}

func (b *YourBatch) reset() error {
	batch, err := b.conn.PrepareBatch(b.ctx, b.insertStatement, driver.WithReleaseConnection())
	if err != nil {
		return err
	}

	b.batch = batch

	return nil
}
