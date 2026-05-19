package std

import (
	"github.com/ClickHouse/clickhouse-go/v2"
	"time"

	_ "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"
)

func BatchInsert() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	conn.SetMaxIdleConns(5)
	if _, err := conn.Exec(`DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS example (
			  Col1 UInt8
			, Col2 String
			, Col3 FixedString(3)
			, Col4 UUID
			, Col5 Map(String, UInt8)
			, Col6 Array(String)
			, Col7 Tuple(String, UInt8, Array(Map(String, String)))
			, Col8 DateTime
		) Engine = Memory
	`)
	if err != nil {
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
	for i := 0; i < 1000; i++ {
		_, err := batch.Exec(
			uint8(42),
			"ClickHouse", "Inc",
			uuid.New(),
			map[string]uint8{"key": 1},             // Map(String, UInt8)
			[]string{"Q", "W", "E", "R", "T", "Y"}, // Array(String)
			[]any{ // Tuple(String, UInt8, Array(Map(String, String)))
				"String Value", uint8(5), []map[string]string{
					map[string]string{"key": "value"},
					map[string]string{"key": "value"},
					map[string]string{"key": "value"},
				},
			},
			time.Now(),
		)
		if err != nil {
			return err
		}
	}
	return scope.Commit()
}
