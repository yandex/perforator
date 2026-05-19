package std

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func QBit() error {
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
	_, err = conn.ExecContext(ctx, "DROP TABLE IF EXISTS example")
	if err != nil {
		return err
	}

	// Create table with QBit column for storing vector embeddings
	// QBit stores vectors in bit-sliced format for efficient vector search
	const ddl = `
		CREATE TABLE example (
			  id UInt32,
			  embedding QBit(Float32, 128)
		) Engine MergeTree() ORDER BY id
		`

	if _, err := conn.ExecContext(ctx, ddl); err != nil {
		return err
	}
	fmt.Println("Table created with QBit column")

	// Insert vectors using transaction
	scope, err := conn.Begin()
	if err != nil {
		return err
	}

	batch, err := scope.PrepareContext(ctx, "INSERT INTO example")
	if err != nil {
		return err
	}

	// Insert 5 sample vectors
	for i := range 5 {
		// Create a sample 128-dimensional vector
		vector := make([]float32, 128)
		for j := range 128 {
			vector[j] = rand.Float32()
		}

		if _, err := batch.ExecContext(ctx, uint32(i), vector); err != nil {
			return err
		}
	}

	if err := scope.Commit(); err != nil {
		return err
	}

	fmt.Println("Inserted 5 vectors")

	// Query vectors back
	rows, err := conn.QueryContext(ctx, "SELECT id, embedding FROM example ORDER BY id")
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("\nRetrieved vectors:")
	for rows.Next() {
		var (
			id        uint32
			embedding []float32
		)
		if err := rows.Scan(&id, &embedding); err != nil {
			return err
		}
		fmt.Printf("ID: %d, Vector dimension: %d, First 5 values: %v\n",
			id, len(embedding), embedding[:5])
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Demonstrate vector search with transposed distance functions
	// Create a query vector
	queryVector := make([]float32, 128)
	for i := range 128 {
		queryVector[i] = rand.Float32()
	}

	fmt.Println("\nPerforming vector similarity search...")

	// Use L2DistanceTransposed for vector similarity search
	// The function is optimized for QBit's bit-sliced storage format
	var searchQuery = `
		SELECT
			id,
			L2DistanceTransposed(embedding, ?::Array(Float32), 32) as distance
		FROM example
		ORDER BY distance ASC
		LIMIT 3
	`

	searchRows, err := conn.QueryContext(ctx, searchQuery, queryVector)
	if err != nil {
		return err
	}
	defer searchRows.Close()

	fmt.Println("\nTop 3 nearest vectors:")
	rank := 1
	for searchRows.Next() {
		var (
			id       uint32
			distance sql.NullFloat64
		)
		if err := searchRows.Scan(&id, &distance); err != nil {
			return err
		}
		if distance.Valid {
			fmt.Printf("%d. ID: %d, L2 Distance: %.4f\n", rank, id, distance.Float64)
		} else {
			fmt.Printf("%d. ID: %d, L2 Distance: NULL\n", rank, id)
		}
		rank++
	}

	return nil
}
