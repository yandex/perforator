package clickhouse_api

import (
	"context"
)

func Exec() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	defer func() {
		conn.Exec(context.Background(), "DROP TABLE example")
	}()
	conn.Exec(context.Background(), `DROP TABLE IF EXISTS example`)
	err = conn.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS example (
			Col1 UInt8,
			Col2 String
		) engine=Memory
	`)
	if err != nil {
		return err
	}
	return conn.Exec(context.Background(), "INSERT INTO example VALUES (1, 'test-1')")
}
