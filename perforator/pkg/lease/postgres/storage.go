package postgres

import (
	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	hasql "golang.yandex/hasql/sqlx"
)

var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// RowLayout names an unqualified table and its lease columns; names are SQL-quoted.
// KeyColumn must be unique; HolderColumn and ExpiresAtColumn must be nullable.
type RowLayout struct {
	Table           string
	KeyColumn       string
	HolderColumn    string
	ExpiresAtColumn string
}

type storageOptions struct {
	layout           RowLayout
	existingRowsOnly bool
}

type StorageOption func(*storageOptions)

// WithRowLayout sets the layout; default: leases(name, holder, expires_at).
func WithRowLayout(layout RowLayout) StorageOption {
	return func(o *storageOptions) { o.layout = layout }
}

// WithExistingRowsOnly disables row creation for caller-managed data.
func WithExistingRowsOnly() StorageOption {
	return func(o *storageOptions) { o.existingRowsOnly = true }
}

// Storage manages row leases, creating missing rows by default.
// Non-lease columns must be nullable or have defaults. Release clears lease fields,
// preserving the row. The configured columns are part of the public contract.
//
// Guard writes by key and token in the same SQL statement; check RowsAffected.
// Also check expiry against clock_timestamp() to reject writes after expiration.
// Recheck row eligibility after acquisition: a previous owner may have finished it.
type Storage struct {
	cluster                       *hasql.Cluster
	table, key, holder, expiresAt string
	existingRowsOnly              bool
}

func NewStorage(cluster *hasql.Cluster, opts ...StorageOption) *Storage {
	options := storageOptions{layout: RowLayout{
		Table: "leases", KeyColumn: "name", HolderColumn: "holder", ExpiresAtColumn: "expires_at",
	}}
	for _, opt := range opts {
		opt(&options)
	}
	quote := func(name string) string { return pgx.Identifier{name}.Sanitize() }
	return &Storage{
		cluster: cluster, table: quote(options.layout.Table), key: quote(options.layout.KeyColumn),
		holder: quote(options.layout.HolderColumn), expiresAt: quote(options.layout.ExpiresAtColumn),
		existingRowsOnly: options.existingRowsOnly,
	}
}
