package clickhouse_api

import (
	"context"
	"time"
)

type row struct {
	Col1       uint64
	Col4       time.Time
	Col2       string
	Col3       []uint8
	ColIgnored string
}

func AppendStruct() error {
	conn, err := GetNativeConnection(nil, nil, nil)
	if err != nil {
		return err
	}
	ctx := context.Background()
	defer func() {
		conn.Exec(ctx, "DROP TABLE example")
	}()
	if err := conn.Exec(ctx, `DROP TABLE IF EXISTS example`); err != nil {
		return err
	}
	if err := conn.Exec(ctx, `
		CREATE TABLE example (
			  Col1 UInt64
			, Col2 String
			, Col3 Array(UInt8)
			, Col4 DateTime
		) Engine = Memory
		`); err != nil {
		return err
	}

	batch, err := conn.PrepareBatch(context.Background(), "INSERT INTO example")
	if err != nil {
		return err
	}
	for i := 0; i < 1_000; i++ {
		err := batch.AppendStruct(&row{
			Col1:       uint64(i),
			Col2:       "Golang SQL database driver",
			Col3:       []uint8{1, 2, 3, 4, 5, 6, 7, 8, 9},
			Col4:       time.Now(),
			ColIgnored: "this will be ignored",
		})
		if err != nil {
			return err
		}
	}
	return batch.Send()
}
