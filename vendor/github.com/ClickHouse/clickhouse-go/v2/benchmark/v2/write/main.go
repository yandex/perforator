package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

const ddl = `
CREATE TABLE benchmark (
	  Col1 UInt64
	, Col2 String
	, Col3 Array(UInt8)
	, Col4 DateTime
) Engine Null
`

func benchmark(conn *sql.DB) error {
	scope, err := conn.Begin()
	if err != nil {
		return err
	}
	{
		batch, err := scope.Prepare("INSERT INTO benchmark")
		if err != nil {
			return err
		}
		for i := 0; i < 1_000_000; i++ {
			_, err := batch.Exec(
				uint64(i),
				"Golang SQL database driver",
				[]uint8{1, 2, 3, 4, 5, 6, 7, 8, 9},
				time.Now(),
			)
			if err != nil {
				return err
			}
		}
		if err := batch.Close(); err != nil {
			return err
		}
	}
	return scope.Commit()
}
func main() {
	conn, err := sql.Open("clickhouse", "clickhouse://127.0.0.1:9000")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := conn.Exec("DROP TABLE IF EXISTS benchmark"); err != nil {
		log.Fatal(err)
	}
	if _, err := conn.Exec(ddl); err != nil {
		log.Fatal(err)
	}
	start := time.Now()
	if err := benchmark(conn); err != nil {
		log.Fatal(err)
	}
	fmt.Println(time.Since(start))
}
