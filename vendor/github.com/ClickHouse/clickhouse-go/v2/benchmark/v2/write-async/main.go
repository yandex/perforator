package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

const ddl = `
CREATE TABLE benchmark_async (
	  Col1 UInt64
	, Col2 String
	, Col3 Array(UInt8)
	, Col4 DateTime
) Engine Null
`

func benchmark(conn clickhouse.Conn) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ctx = clickhouse.Context(ctx, clickhouse.WithAsync(true))
	for i := 0; i < 10_000; i++ {
		err := conn.Exec(ctx, fmt.Sprintf(`INSERT INTO benchmark_async VALUES (
			%d, '%s', [1, 2, 3, 4, 5, 6, 7, 8, 9], now()
		)`, i, "Golang SQL database driver"), false)
		if err != nil {
			return err
		}
	}
	return nil
}

func main() {
	var (
		ctx       = context.Background()
		conn, err = clickhouse.Open(&clickhouse.Options{
			Addr: []string{"127.0.0.1:9000"},
			Auth: clickhouse.Auth{
				Database: "default",
				Username: "default",
				Password: "",
			},
			//Debug:           true,
			DialTimeout:     time.Second,
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: time.Hour,
		})
	)
	if err != nil {
		log.Fatal(err)
	}
	if err := conn.Exec(ctx, "DROP TABLE IF EXISTS benchmark_async"); err != nil {
		log.Fatal(err)
	}
	if err := conn.Exec(ctx, ddl); err != nil {
		log.Fatal(err)
	}
	start := time.Now()
	if err := benchmark(conn); err != nil {
		log.Fatal(err)
	}
	fmt.Println(time.Since(start))
}
