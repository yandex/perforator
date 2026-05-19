package clickhouse_api

import (
	"context"
	"fmt"
	"github.com/ClickHouse/clickhouse-go/v2"
	clickhouse_tests "github.com/ClickHouse/clickhouse-go/v2/tests"
)

func QueryWithParameters() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}

	if !clickhouse_tests.CheckMinServerServerVersion(conn, 22, 8, 0) {
		return nil
	}

	chCtx := clickhouse.Context(context.Background(), clickhouse.WithParameters(clickhouse.Parameters{
		"str":      "hello",
		"array":    "['a', 'b', 'c']",
		"column":   "number",
		"database": "system",
		"table":    "numbers",
	}))

	row := conn.QueryRow(chCtx, "SELECT {column:Identifier} v, {str:String} s, {array:Array(String)} a FROM {database:Identifier}.{table:Identifier} LIMIT 1 OFFSET 100")
	var (
		col1 uint64
		col2 string
		col3 []string
	)
	if err := row.Scan(&col1, &col2, &col3); err != nil {
		return err
	}
	fmt.Printf("row: col1=%d, col2=%s, col3=%s\n", col1, col2, col3)
	return nil
}
