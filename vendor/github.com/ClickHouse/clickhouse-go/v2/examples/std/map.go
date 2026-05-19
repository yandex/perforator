package std

import (
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func MapInsertRead() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	const ddl = `
		CREATE TABLE example (
			  Col1 Map(String, UInt64)
			, Col2 Map(String, UInt64)
			, Col3 Map(String, UInt64)
			, Col4 Array(Map(String, String))
			, Col5 Map(LowCardinality(String), LowCardinality(String))
		) Engine Memory
		`
	conn.Exec("DROP TABLE example")
	defer func() {
		conn.Exec("DROP TABLE example")
	}()
	if _, err := conn.Exec(ddl); err != nil {
		return err
	}
	scope, err := conn.Begin()
	if err != nil {
		return err
	}
	batch, err := scope.Prepare("INSERT INTO example")
	if err != nil {
		return err
	}
	var (
		col1Data = map[string]uint64{
			"key_col_1_1": 1,
			"key_col_1_2": 2,
		}
		col2Data = map[string]uint64{
			"key_col_2_1": 10,
			"key_col_2_2": 20,
		}
		col3Data = map[string]uint64{}
		col4Data = []map[string]string{
			{"A": "B"},
			{"C": "D"},
		}
		col5Data = map[string]string{
			"key_col_5_1": "100",
			"key_col_5_2": "200",
		}
	)
	if _, err := batch.Exec(col1Data, col2Data, col3Data, col4Data, col5Data); err != nil {
		return err
	}
	if err = scope.Commit(); err != nil {
		return err
	}
	var (
		col1 any
		col2 map[string]uint64
		col3 map[string]uint64
		col4 []map[string]string
		col5 map[string]string
	)
	if err := conn.QueryRow("SELECT * FROM example").Scan(&col1, &col2, &col3, &col4, &col5); err != nil {
		return err
	}
	fmt.Printf("col1=%v, col2=%v, col3=%v, col4=%v, col5=%v", col1, col2, col3, col4, col5)
	return nil
}
