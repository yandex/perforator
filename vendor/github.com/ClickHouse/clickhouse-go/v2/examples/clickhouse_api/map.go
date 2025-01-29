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
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column"
)

func MapInsertRead() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	conn.Exec(context.Background(), "DROP TABLE IF EXISTS example")
	err = conn.Exec(ctx, `
		CREATE TABLE example (
			  Col1 Map(String, UInt64)
			, Col2 Map(String, Array(String))
			, Col3 Map(String, Map(String,UInt64))
		) Engine Memory
	`)
	if err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	var i int64
	for i = 0; i < 10; i++ {
		err := batch.Append(
			map[string]uint64{strconv.Itoa(int(i)): uint64(i)},
			map[string][]string{strconv.Itoa(int(i)): {strconv.Itoa(int(i)), strconv.Itoa(int(i + 1)), strconv.Itoa(int(i + 2)), strconv.Itoa(int(i + 3))}},
			map[string]map[string]uint64{strconv.Itoa(int(i)): {strconv.Itoa(int(i)): uint64(i)}},
		)
		if err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	var (
		col1 map[string]uint64
		col2 map[string][]string
		col3 map[string]map[string]uint64
	)
	rows, err := conn.Query(ctx, "SELECT * FROM example")
	if err != nil {
		return err
	}
	for rows.Next() {
		if err := rows.Scan(&col1, &col2, &col3); err != nil {
			return err
		}
		fmt.Printf("row: col1=%v, col2=%v, col3=%v\n", col1, col2, col3)
	}
	rows.Close()
	return rows.Err()
}

func IterableOrderedMapInsertRead() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	conn.Exec(context.Background(), "DROP TABLE IF EXISTS example")
	err = conn.Exec(ctx, `
		CREATE TABLE example (
			  Col1 Map(String, String)
		) Engine Memory
	`)
	if err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	var i int64
	for i = 0; i < 10; i++ {
		om := NewOrderedMap()
		kv1 := strconv.Itoa(int(i))
		kv2 := strconv.Itoa(int(i + 1))
		om.Put(kv1, kv1)
		om.Put(kv2, kv2)
		err := batch.Append(om)
		if err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	rows, err := conn.Query(ctx, "SELECT * FROM example")
	if err != nil {
		return err
	}
	for rows.Next() {
		var col1 OrderedMap
		if err := rows.Scan(&col1); err != nil {
			return err
		}
		fmt.Printf("row: col1=%v\n", col1)
	}
	rows.Close()
	return rows.Err()
}

// OrderedMap is a simple (non thread safe) ordered map
type OrderedMap struct {
	Keys   []any
	Values []any
}

func NewOrderedMap() column.IterableOrderedMap {
	return &OrderedMap{}
}

func (om *OrderedMap) Put(key any, value any) {
	om.Keys = append(om.Keys, key)
	om.Values = append(om.Values, value)
}

func (om *OrderedMap) Iterator() column.MapIterator {
	return NewOrderedMapIterator(om)
}

type OrderedMapIter struct {
	om        *OrderedMap
	iterIndex int
}

func NewOrderedMapIterator(om *OrderedMap) column.MapIterator {
	return &OrderedMapIter{om: om, iterIndex: -1}
}

func (i *OrderedMapIter) Next() bool {
	i.iterIndex++
	return i.iterIndex < len(i.om.Keys)
}

func (i *OrderedMapIter) Key() any {
	return i.om.Keys[i.iterIndex]
}

func (i *OrderedMapIter) Value() any {
	return i.om.Values[i.iterIndex]
}
