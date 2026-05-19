package std

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/ext"
)

func ExternalData() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}

	table1, err := ext.NewTable("external_table_1",
		ext.Column("col1", "UInt8"),
		ext.Column("col2", "String"),
		ext.Column("col3", "DateTime"),
	)
	if err != nil {
		return err
	}

	for i := 0; i < 10; i++ {
		if err = table1.Append(uint8(i), fmt.Sprintf("value_%d", i), time.Now()); err != nil {
			return err
		}
	}

	table2, err := ext.NewTable("external_table_2",
		ext.Column("col1", "UInt8"),
		ext.Column("col2", "String"),
		ext.Column("col3", "DateTime"),
	)

	for i := 0; i < 10; i++ {
		table2.Append(uint8(i), fmt.Sprintf("value_%d", i), time.Now())
	}
	ctx := clickhouse.Context(context.Background(),
		clickhouse.WithExternalTable(table1, table2),
	)
	rows, err := conn.QueryContext(ctx, "SELECT * FROM external_table_1")
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			col1 uint8
			col2 string
			col3 time.Time
		)
		rows.Scan(&col1, &col2, &col3)
		fmt.Printf("col1=%d, col2=%s, col3=%v\n", col1, col2, col3)
	}

	var count uint64
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM external_table_1").Scan(&count); err != nil {
		return err
	}
	fmt.Printf("external_table_1: %d\n", count)
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM external_table_2").Scan(&count); err != nil {
		return err
	}
	fmt.Printf("external_table_2: %d\n", count)
	if err := conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM (SELECT * FROM external_table_1 UNION ALL SELECT * FROM external_table_2)").Scan(&count); err != nil {
		return err
	}
	fmt.Printf("external_table_1 UNION external_table_2: %d\n", count)
	return nil
}
