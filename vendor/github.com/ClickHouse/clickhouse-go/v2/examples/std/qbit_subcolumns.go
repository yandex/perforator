package std

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func QBitSubcolumns() error {
	conn, err := GetStdOpenDBConnection(clickhouse.Native, nil, nil, nil)
	if err != nil {
		return err
	}
	if !CheckMinServerVersion(conn, 25, 10, 0) {
		fmt.Print("unsupported clickhouse version for QBit type")
		return nil
	}
	ctx := context.Background()
	ctx = clickhouse.Context(ctx, clickhouse.WithSettings(clickhouse.Settings{
		// QBit is an experimental feature in ClickHouse
		"allow_experimental_qbit_type": 1,
	}))

	_, err = conn.ExecContext(ctx, "DROP TABLE IF EXISTS example_subcolumns")
	if err != nil {
		return err
	}

	// Create table with QBit column
	const ddl = `
		CREATE TABLE example_subcolumns (
			  id UInt8,
			  vec QBit(Float32, 4)
		) Engine MergeTree() ORDER BY id
		`

	if _, err := conn.ExecContext(ctx, ddl); err != nil {
		return err
	}
	fmt.Println("Table created with QBit column")

	// Insert vectors with special float values to demonstrate bit patterns
	// Float32 has 32 bits: 1 sign bit + 8 exponent bits + 23 mantissa bits
	// Bit numbering in QBit: .1 is MSB (sign bit), .32 is LSB
	scope, err := conn.Begin()
	if err != nil {
		return err
	}

	batch, err := scope.PrepareContext(ctx, "INSERT INTO example_subcolumns")
	if err != nil {
		return err
	}

	// In Go, compiler won't let you use -0.0 constant
	pZero := float32(0.0)
	nZero := -pZero

	// Insert vectors with positive and negative zeros to show sign bit difference
	vectors := []struct {
		id   uint8
		vec  []float32
		desc string
	}{
		{1, []float32{1.0, -1.0, 0.0, nZero}, "Mixed signs"},
		{2, []float32{2.5, -2.5, 3.5, -3.5}, "Positive and negative"},
		{3, []float32{0.0, 0.0, 0.0, 0.0}, "All positive zeros"},
		{4, []float32{nZero, nZero, nZero, nZero}, "All negative zeros"},
	}

	for _, v := range vectors {
		if _, err := batch.ExecContext(ctx, v.id, v.vec); err != nil {
			return err
		}
	}

	if err := scope.Commit(); err != nil {
		return err
	}

	fmt.Println("Inserted vectors with special float values")

	// Access subcolumns to examine bit patterns
	// .1 is the sign bit (MSB of Float32)
	// .2-.9 are exponent bits
	// .10-.32 are mantissa bits
	fmt.Println("\nAccessing QBit subcolumns:")
	fmt.Println("Note: vec.1 is the sign bit (1 = negative, 0 = positive)")
	fmt.Println()

	query := `
		SELECT
			id,
			vec,
			bin(vec.1) as sign_bits,
			bin(vec.2) as exponent_bit1,
			bin(vec.9) as exponent_bit8
		FROM example_subcolumns
		ORDER BY id
	`

	rows, err := conn.QueryContext(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id           uint8
			vec          []float32
			signBits     string
			exponentBit1 string
			exponentBit8 string
		)
		if err := rows.Scan(&id, &vec, &signBits, &exponentBit1, &exponentBit8); err != nil {
			return err
		}

		fmt.Printf("ID %d: %v\n", id, vec)
		fmt.Printf("  Sign bits (.1):      %s  (1=negative, 0=positive)\n", signBits)
		fmt.Printf("  Exponent bit 1 (.2): %s\n", exponentBit1)
		fmt.Printf("  Exponent bit 8 (.9): %s\n", exponentBit8)
		fmt.Println()
	}

	return nil
}
