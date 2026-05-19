package clickhouse_api

import (
	"context"
	"fmt"
	"time"
)

func SelectStruct() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS example (
			Col1 UInt8,
			Col2 String,
			Col3 DateTime
		) engine=Memory
	`)
	defer func() {
		conn.Exec(context.Background(), "DROP TABLE example")
	}()
	if err != nil {
		return err
	}
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example (Col1, Col2, Col3)")
	if err != nil {
		return err
	}
	for i := 0; i < 100; i++ {
		if err := batch.Append(uint8(i), fmt.Sprintf("value_%d", i), time.Now().Add(time.Duration(i)*time.Second)); err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}

	var result []struct {
		Col1           uint8
		Col2           string
		ColumnWithName time.Time `ch:"Col3"`
	}

	if err = conn.Select(ctx, &result, "SELECT Col1, Col2, Col3 FROM example"); err != nil {
		return err
	}

	for _, v := range result {
		fmt.Printf("row: col1=%d, col2=%s, col3=%s\n", v.Col1, v.Col2, v.ColumnWithName)
	}

	return nil
}
