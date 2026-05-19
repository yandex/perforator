package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

const ddl = `
CREATE TABLE benchmark (
	  Col1 UInt64
	, Col2 String
	, Col3 Array(UInt8)
	, Col4 DateTime
) Engine Null
`

type row struct {
	Col1 uint64
	Col4 time.Time
	Col2 string
	Col3 []uint8
}

func benchmark(conn clickhouse.Conn) error {
	batch, err := conn.PrepareBatch(context.Background(), "INSERT INTO benchmark")
	if err != nil {
		return err
	}
	for i := 0; i < 1_000_000; i++ {
		err := batch.AppendStruct(&row{
			Col1: uint64(i),
			Col2: "Golang SQL database driver",
			Col3: []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9},
			Col4: time.Now(),
		})
		if err != nil {
			return err
		}
	}
	return batch.Send()
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
	if err := conn.Exec(ctx, "DROP TABLE IF EXISTS benchmark"); err != nil {
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
