package std

import "github.com/ClickHouse/clickhouse-go/v2"

func Exec() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	defer func() {
		conn.Exec("DROP TABLE example")
	}()
	conn.Exec(`DROP TABLE IF EXISTS example`)
	_, err = conn.Exec(`
		CREATE TABLE IF NOT EXISTS example (
			Col1 UInt8,
			Col2 String
		) engine=Memory
	`)
	if err != nil {
		return err
	}
	_, err = conn.Exec("INSERT INTO example VALUES (1, 'test-1')")
	return err
}
