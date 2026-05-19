package clickhouse_api

import (
	"context"
	"fmt"
	"strconv"
)

func ArrayInsertRead() error {
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
		CREATE TABLE IF NOT EXISTS example (
			  Col1 Array(String)
			, Col2 Array(Array(Int64))
		) Engine = Memory
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
			[]string{strconv.Itoa(int(i)), strconv.Itoa(int(i + 1)), strconv.Itoa(int(i + 2)), strconv.Itoa(int(i + 3))},
			[][]int64{{i, i + 1}, {i + 2, i + 3}, {i + 4, i + 5}},
		)
		if err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	var (
		col1 []string
		col2 [][]int64
	)
	rows, err := conn.Query(ctx, "SELECT * FROM example")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&col1, &col2); err != nil {
			return err
		}
		fmt.Printf("row: col1=%v, col2=%v\n", col1, col2)
	}
	return rows.Err()
}
