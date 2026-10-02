package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"

	leasepkg "github.com/yandex/perforator/perforator/pkg/lease"
)

// ForKey returns a reusable lease handle without database access.
func ForKey[K ~string | ~int64](s *Storage, key K) leasepkg.Lease {
	return &lease[K]{storage: s, key: key}
}

type lease[K ~string | ~int64] struct {
	storage *Storage
	key     K
}

func (l *lease[K]) openPredicate(holder string) squirrel.Sqlizer {
	s := l.storage
	return squirrel.And{
		squirrel.Eq{s.key: l.key, s.holder: holder},
		squirrel.NotEq{s.expiresAt: nil},
	}
}

func leaseExpiry(ttl time.Duration) squirrel.Sqlizer {
	return squirrel.Expr("clock_timestamp() + ? * interval '1 microsecond'", ttl.Microseconds())
}

func execLeaseQuery(ctx context.Context, db *sqlx.DB, query squirrel.Sqlizer) (bool, error) {
	sql, args, err := query.ToSql()
	if err != nil {
		return false, err
	}
	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return false, err
	}
	rows, err := result.RowsAffected()
	return rows == 1, err
}

func (l *lease[K]) Acquire(ctx context.Context, holder string, ttl time.Duration) (bool, error) {
	if ttl.Microseconds() <= 0 {
		return false, fmt.Errorf("lease TTL must be at least one microsecond")
	}
	s := l.storage
	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return false, err
	}
	expiresAt := "lease_row." + s.expiresAt
	available := squirrel.Or{squirrel.Eq{expiresAt: nil}, squirrel.Expr(expiresAt + " <= clock_timestamp()")}
	var query squirrel.Sqlizer = psql.Update(s.table+" AS lease_row").
		Set(s.holder, holder).Set(s.expiresAt, leaseExpiry(ttl)).
		Where(squirrel.Eq{s.key: l.key}).Where(available)
	if !s.existingRowsOnly {
		// Recompute expiry after conflict waits; EXCLUDED expiry may be stale.
		query = psql.Insert(s.table+" AS lease_row").Columns(s.key, s.holder, s.expiresAt).
			Values(l.key, holder, leaseExpiry(ttl)).
			Suffix(fmt.Sprintf("ON CONFLICT (%s) DO UPDATE SET %s = EXCLUDED.%s, %s = ? WHERE ?",
				s.key, s.holder, s.holder, s.expiresAt), leaseExpiry(ttl), available)
	}
	return execLeaseQuery(ctx, primary.DBx(), query)
}

// Renew treats NULL expiry as closed, even if the token remains.
func (l *lease[K]) Renew(ctx context.Context, holder string, ttl time.Duration) (bool, error) {
	if ttl.Microseconds() <= 0 {
		return false, fmt.Errorf("lease TTL must be at least one microsecond")
	}
	primary, err := l.storage.cluster.WaitForPrimary(ctx)
	if err != nil {
		return false, err
	}
	s := l.storage
	return execLeaseQuery(ctx, primary.DBx(), psql.Update(s.table).
		Set(s.expiresAt, squirrel.Expr("GREATEST("+s.expiresAt+", ?)", leaseExpiry(ttl))).
		Where(l.openPredicate(holder)))
}

// Release clears matching ownership, preserving the row.
func (l *lease[K]) Release(ctx context.Context, holder string) error {
	primary, err := l.storage.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}
	s := l.storage
	_, err = execLeaseQuery(ctx, primary.DBx(), psql.Update(s.table).
		Set(s.holder, nil).Set(s.expiresAt, nil).Where(l.openPredicate(holder)))
	return err
}
