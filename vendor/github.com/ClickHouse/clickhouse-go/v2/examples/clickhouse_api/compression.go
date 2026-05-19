package clickhouse_api

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	"strconv"
)

func Compress() error {
	env, err := GetNativeTestEnvironment()
	if err != nil {
		return err
	}
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.Port)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionZSTD,
		},
		MaxOpenConns: 1,
	})
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	conn.Exec(context.Background(), "DROP TABLE IF EXISTS example")
	if err = conn.Exec(ctx, `
		CREATE TABLE example (
			  Col1 Array(String)
		) Engine Memory
		`); err != nil {
		return err
	}
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}
	for i := 0; i < 1000; i++ {
		if err := batch.Append([]string{strconv.Itoa(i), strconv.Itoa(i + 1), strconv.Itoa(i + 2), strconv.Itoa(i + 3)}); err != nil {
			return err
		}
	}
	if err := batch.Send(); err != nil {
		return err
	}
	return nil
}
