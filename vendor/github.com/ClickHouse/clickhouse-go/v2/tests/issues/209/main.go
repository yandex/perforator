package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
)

func getClickhouseClient() driver.Conn {
	conn, _ := clickhouse_tests.GetConnectionWithOptions(&clickhouse.Options{
		Addr: []string{"127.0.0.1:9000"},
		Auth: clickhouse.Auth{
			Database: "",
			Username: "",
			Password: "",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout:     5 * time.Second,
		ConnMaxLifetime: 15 * time.Second,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		// Debug: true,
	})

	return conn
}

func main() {
	conn := getClickhouseClient()
	http.HandleFunc("/test", func(rw http.ResponseWriter, r *http.Request) {
		var result []struct {
			Test string `ch:"test"`
		}
		sql := `SELECT 'test' AS test FROM system.numbers LIMIT 10`
		if response := conn.Select(context.Background(), &result, sql); response != nil {
			fmt.Println(response.Error())
		}
		fmt.Println(result, conn.Stats())
	})
	http.ListenAndServe(":8080", nil)
}
