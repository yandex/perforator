package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
)

func benchmarkRead(conn *sql.DB) error {
	rows, err := conn.Query(`
SELECT
	number
	, randomString(25)
	, array(1, 2, 3, 4, 5)
	, now()
FROM system.numbers LIMIT 1000000
`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var (
			col1 uint64
			col2 string
			col3 []uint8
			col4 time.Time
		)
		if err := rows.Scan(&col1, &col2, &col3, &col4); err != nil {
			return err
		}
	}
	return nil
}

func benchmarkString(conn *sql.DB) error {
	rows, err := conn.Query(`SELECT toString(number) FROM numbers(500000000)`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var (
			col1 string
		)
		if err := rows.Scan(&col1); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	conn, err := sql.Open("clickhouse", "clickhouse://127.0.0.1:9000")
	if err != nil {
		log.Fatal(err)
	}
	start := time.Now()
	if err := benchmarkRead(conn); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("benchmarkRead: %v\n", time.Since(start))
	start = time.Now()
	if err := benchmarkString(conn); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("benchmarkString: %v\n", time.Since(start))
}
