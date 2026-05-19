package clickhouse_api

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2/lib/column/orderedmap"
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
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&col1, &col2, &col3); err != nil {
			return err
		}
		fmt.Printf("row: col1=%v, col2=%v, col3=%v\n", col1, col2, col3)
	}

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
		om := &orderedmap.Map[string, string]{}
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
	defer rows.Close()

	for rows.Next() {
		var col1 orderedmap.Map[string, string]
		if err := rows.Scan(&col1); err != nil {
			return err
		}
		fmt.Printf("row: col1=%v\n", col1)
	}

	return rows.Err()
}
