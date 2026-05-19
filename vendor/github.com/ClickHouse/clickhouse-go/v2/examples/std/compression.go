package std

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func CompressOpenDB() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	conn := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", env.Host, env.HttpPort)},
		Auth: clickhouse.Auth{
			Database: env.Database,
			Username: env.Username,
			Password: env.Password,
		},
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionBrotli,
			Level:  5,
		},
		Protocol: clickhouse.HTTP,
	})
	defer func() {
		conn.Exec("DROP TABLE example")
	}()
	conn.Exec("DROP TABLE IF EXISTS example")
	if _, err := conn.Exec(`
		CREATE TABLE example (
			  Col1 Array(String)
			, Col2 UInt64
		) Engine Memory
		`); err != nil {
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
		if _, err := batch.Exec(
			[]string{strconv.Itoa(i), strconv.Itoa(i + 1), strconv.Itoa(i + 2), strconv.Itoa(i + 3)},
			uint64(i),
		); err != nil {
			return err
		}
	}
	if err := scope.Commit(); err != nil {
		return err
	}
	return nil
}

func CompressOpen() error {
	env, err := GetStdTestEnvironment()
	if err != nil {
		return err
	}
	// note compress=gzip&compress_level=5
	conn, err := sql.Open("clickhouse", fmt.Sprintf("http://%s:%d?username=%s&password=%s&compress=gzip&compress_level=5", env.Host, env.HttpPort, env.Username, env.Password))
	defer func() {
		conn.Exec("DROP TABLE example")
	}()
	conn.Exec("DROP TABLE IF EXISTS example")
	if _, err := conn.Exec(`
		CREATE TABLE example (
			  Col1 Array(String)
			, Col2 UInt64
		) Engine Memory
		`); err != nil {
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
		if _, err := batch.Exec(
			[]string{strconv.Itoa(i), strconv.Itoa(i + 1), strconv.Itoa(i + 2), strconv.Itoa(i + 3)},
			uint64(i),
		); err != nil {
			return err
		}
	}
	if err := scope.Commit(); err != nil {
		return err
	}
	return nil
}
