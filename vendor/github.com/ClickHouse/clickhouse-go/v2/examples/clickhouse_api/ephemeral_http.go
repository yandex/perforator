package clickhouse_api

import (
	"context"
	"fmt"
)

func EphemeralColumnHTTP() error {
	conn, err := GetHTTPConnection("http-ephemeral-column", nil, nil, nil)
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

	if err := conn.Exec(ctx, ddl); err != nil {
		return err
	}
	fmt.Println("Created table 'test'")

	i := `INSERT INTO test (id, unhexed) VALUES (1, '5a90b714');`

	if err := conn.Exec(ctx, i); err != nil {
		return err
	}
	fmt.Println("Inserted rows into 'test'")

	query := `
	SELECT
		id,
		hexed,
		hex(hexed)
	FROM test;`

	rows, err := conn.Query(ctx, query)
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
