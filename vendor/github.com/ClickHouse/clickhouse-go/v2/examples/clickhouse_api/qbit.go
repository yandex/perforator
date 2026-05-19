package clickhouse_api

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/ClickHouse/clickhouse-go/v2"
)

func QBit() error {
	conn, err := GetNativeConnection(nil, nil, nil)
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

	conn.Exec(ctx, "DROP TABLE IF EXISTS example")

	// Create table with QBit column for storing vector embeddings
	// QBit stores vectors in bit-sliced format for efficient vector search
	const ddl = `
		CREATE TABLE example (
			  id UInt32,
			  embedding QBit(Float32, 128)
		) Engine MergeTree() ORDER BY id
		`

	if err := conn.Exec(ctx, ddl); err != nil {
		return err
	}
	fmt.Println("Table created with QBit column")

	// Insert vectors into the table
	batch, err := conn.PrepareBatch(ctx, "INSERT INTO example")
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

		if err := batch.Append(uint32(i), vector); err != nil {
			return err
		}
	}

	fmt.Printf("Prepared %d vectors for insertion\n", batch.Rows())
	if err := batch.Send(); err != nil {
		return err
	}

	fmt.Printf("Inserted %d vectors\n", batch.Rows())

	// Query vectors back
	rows, err := conn.Query(ctx, "SELECT id, embedding FROM example ORDER BY id")
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

	searchRows, err := conn.Query(ctx, searchQuery, queryVector)
	if err != nil {
		return err
	}
	defer searchRows.Close()

	fmt.Println("\nTop 3 nearest vectors:")
	for searchRows.Next() {
		var (
			id       uint32
			distance float64
		)
		if err := searchRows.Scan(&id, &distance); err != nil {
			return err
		}
		fmt.Printf("ID: %d, L2 Distance: %.4f\n", id, distance)
	}

	return nil
}
