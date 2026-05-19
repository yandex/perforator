package clickhouse_api

import (
	"context"
	"fmt"
)

type customStr string

func (s *customStr) Scan(src any) error {
	if t, ok := src.(string); ok {
		*s = customStr(t)
		return nil
	}
	return fmt.Errorf("cannot scan %T into customStr", src)
}

func (s customStr) String() string {
	return string(s)
}

func CustomTypes() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(context.Background(), "DROP TABLE example")
	}()
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	err = conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS example (
			  Col1 String,
			  Col2 Enum ('hello'   = 1,  'world' = 2)
		) Engine = Memory
	`)
	if err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}

	err = batch.Append(
		customStr("A"), customStr("hello"),
	)
	if err != nil {
		return err
	}

	err = batch.Send()
	if err != nil {
		return err
	}

	var (
		col1 customStr
		col2 customStr
	)

	if err = conn.QueryRow(ctx, "SELECT * FROM example").Scan(&col1, &col2); err != nil {
		return err
	}
	fmt.Printf("col1=%v (T=%T), col2=%v (T=%T)\n", col1, col1, col2, col2)
	return nil
}
