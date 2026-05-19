package clickhouse_api

import (
	"context"
	"fmt"
	"time"
)

func ScanStruct() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}

	const ddl = `
	CREATE TABLE example (
		  Col1 Int64
		, Col2 FixedString(2)
		, Col3 Map(String, Int64)
		, Col4 Array(Int64)
		, Col5 DateTime64(3)
	) Engine Memory
	`
	defer func() {
		conn.Exec(context.Background(), "DROP TABLE example")
	}()
	if err := conn.Exec(context.Background(), "DROP TABLE IF EXISTS example"); err != nil {
		return err
	}
	if err := conn.Exec(context.Background(), ddl); err != nil {
		return err
	}
	batch, err := conn.PrepareBatch(context.Background(), "INSERT INTO example")
	if err != nil {
		return err
	}
	for i := int64(0); i < 1000; i++ {
		err := batch.Append(
			i%10,
			"CH",
			map[string]int64{
				"key": i,
			},
			[]int64{i, i + 1, i + 2},
			time.Now(),
		)
		if err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}

	var result struct {
		Col1  int64
		Count uint64 `ch:"count"`
	}
	if err := conn.QueryRow(context.Background(), "SELECT Col1, COUNT() AS count FROM example WHERE Col1 = 5 GROUP BY Col1").ScanStruct(&result); err != nil {
		return err
	}
	fmt.Println("result", result)
	return nil
}
