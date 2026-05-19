package main

import (
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests/std"
)

func main() {
	conn := clickhouse_tests.GetConnectionWithOptions(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		//Debug: true,
	})
	if err := conn.Ping(); err != nil {
		fmt.Printf("1: %v\n", err)
	}
	row := conn.QueryRow("SELECT 1")
	var one int
	if err := row.Scan(&one); err != nil {
		fmt.Printf("2: %v\n", err)
	}
	fmt.Printf("3: %v\n", one)
	if err := conn.Close(); err != nil {
		fmt.Printf("4: %v\n", err)
	}
	fmt.Printf("5\n")
}
