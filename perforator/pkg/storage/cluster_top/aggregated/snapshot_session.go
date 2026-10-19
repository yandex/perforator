package aggregated

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// snapshotSession owns one physical native connection and its temporary tables.
type snapshotSession struct {
	conn *sql.Conn
	pool *sql.DB
}

type snapshotRowScanner interface {
	Scan(...any) error
}

func (s *snapshotSession) Exec(ctx context.Context, query string, args ...any) error {
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func (s *snapshotSession) QueryRow(ctx context.Context, query string, args ...any) snapshotRowScanner {
	return s.conn.QueryRowContext(ctx, query, args...)
}

func (s *snapshotSession) Close() error {
	// Returning the connection alone would keep the session alive in its pool.
	return errors.Join(s.conn.Close(), s.pool.Close())
}

// Failover is allowed only before the rebuild starts. All subsequent operations
// use one pinned session; closing it also removes the temporary staging table.
func (s *SnapshotStorage) openSession(ctx context.Context, timeout time.Duration) (*snapshotSession, error) {
	start := rand.IntN(len(s.options.Addr))
	var errs []error
	for i := range s.options.Addr {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		options := s.sessionOptions(s.options.Addr[(start+i)%len(s.options.Addr)], timeout)
		pool := clickhouse.OpenDB(&options)
		conn, err := pool.Conn(ctx)
		if err == nil {
			err = conn.PingContext(ctx)
			if err == nil {
				return &snapshotSession{conn: conn, pool: pool}, nil
			}
			_ = conn.Close()
		}
		_ = pool.Close()
		errs = append(errs, err)
	}
	return nil, fmt.Errorf("no healthy replica for snapshot rebuild: %w", errors.Join(errs...))
}

func (s *SnapshotStorage) sessionOptions(address string, timeout time.Duration) clickhouse.Options {
	options := *s.options
	options.Addr = []string{address}
	options.MaxOpenConns = 1
	options.MaxIdleConns = 1
	options.ReadTimeout = timeout
	// The driver checks connection age before every statement, even while pinned.
	// Its default one-hour lifetime must not interrupt a longer rebuild.
	options.ConnMaxLifetime = timeout
	return options
}
