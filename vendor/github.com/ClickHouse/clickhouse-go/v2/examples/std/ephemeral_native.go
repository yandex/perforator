package std

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func EphemeralColumnNative() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("Failed to connect: %v\n", err)
	}

	//NOTE: never forget to close the connection at the end.
	defer conn.Close()

	ctx := context.Background()
	ddl := `
	CREATE OR REPLACE TABLE test
	(
		id UInt64,
		unhexed String EPHEMERAL,
		hexed FixedString(4) DEFAULT unhex(unhexed)
	)
	ENGINE = MergeTree
	ORDER BY id;`

	_, err = conn.ExecContext(ctx, ddl)
	if err != nil {
		return err
	}
	fmt.Println("Created table 'test'")

	i := `INSERT INTO test (id, unhexed) VALUES (1, '5a90b714');`

	_, err = conn.ExecContext(ctx, i)
	if err != nil {
		return err
	}
	fmt.Println("Inserted rows into 'test'")

	query := `
	SELECT
		id,
		hexed,
		hex(hexed)
	FROM test;`

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return err
	}

	fmt.Println("Query Results:")
	for rows.Next() {
		var (
			id  uint64
			un  string
			hex string
		)
		if err := rows.Scan(&id, &un, &hex); err != nil {
			panic(err)
		}
		fmt.Println("id: ", id, "un: ", un, "hex: ", hex)
	}
	return nil
}
