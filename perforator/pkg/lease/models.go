package lease

import (
	"context"
	"time"
)

type StorageType string

const (
	Postgres StorageType = "postgres"
)

// Lease addresses one resource; each acquisition requires a fresh token.
// All operations must honor context cancellation.
type Lease interface {
	// Acquire returns false if the resource is busy or unavailable.
	Acquire(ctx context.Context, token string, ttl time.Duration) (bool, error)
	// Renew only extends expiry, including after expiration while still owned.
	// It returns false for a lost or explicitly released lease.
	Renew(ctx context.Context, token string, ttl time.Duration) (bool, error)
	// Release does nothing if the lease is absent or belongs to another token.
	Release(ctx context.Context, token string) error
}
