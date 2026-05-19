package clickhouse_api

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"time"
)

func BindParameters() error {
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
			Col1 UInt32,
			Col2 String,
			Col3 DateTime
		) engine=Memory
	`)
	if err != nil {
		return err
	}
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example (Col1, Col2, Col3)")
	if err != nil {
		return err
	}
	now := time.Now()
	for i := 0; i < 1000; i++ {
		if err := batch.Append(uint32(i), fmt.Sprintf("value_%d", i), now.Add(time.Duration(i)*time.Second)); err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	var count uint64
	// positional bind
	if err = conn.QueryRow(ctx, "SELECT count() FROM example WHERE Col1 >= ? AND Col3 < ?", 500, now.Add(time.Duration(750)*time.Second)).Scan(&count); err != nil {
		return err
	}
	// 250
	fmt.Printf("Positional bind count: %d\n", count)
	// numeric bind
	if err = conn.QueryRow(ctx, "SELECT count() FROM example WHERE Col1 <= $2 AND Col3 > $1", now.Add(time.Duration(150)*time.Second), 250).Scan(&count); err != nil {
		return err
	}
	// 100
	fmt.Printf("Numeric bind count: %d\n", count)
	// named bind
	if err = conn.QueryRow(ctx, "SELECT count() FROM example WHERE Col1 <= @col1 AND Col3 > @col3", clickhouse.Named("col1", 100), clickhouse.Named("col3", now.Add(time.Duration(50)*time.Second))).Scan(&count); err != nil {
		return err
	}
	// 50
	fmt.Printf("Named bind count: %d\n", count)
	return nil
}
