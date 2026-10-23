package aggregated

import (
	"context"
	"fmt"
	"strings"
	"time"

	clickhousego "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/google/uuid"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const clusterTopServingTable = "cluster_top_by_function_v3_serving"

// SnapshotConfig bounds the entire rebuild, not individual retry attempts.
type SnapshotConfig struct {
	Timeout                       time.Duration `yaml:"timeout"`
	MaxThreads                    uint64        `yaml:"max_threads"`
	MaxMemoryUsage                uint64        `yaml:"max_memory_usage"` // Per-query memory budget in bytes.
	MaxBytesBeforeExternalGroupBy uint64        `yaml:"max_bytes_before_external_group_by"`
}

func (c *SnapshotConfig) FillDefault() {
	if c.Timeout == 0 {
		c.Timeout = time.Hour
	}
	if c.MaxThreads == 0 {
		c.MaxThreads = 2
	}
	if c.MaxMemoryUsage == 0 {
		c.MaxMemoryUsage = 16 << 30 // 16 GiB.
	}
	if c.MaxBytesBeforeExternalGroupBy == 0 {
		// Leave headroom for merging aggregation states and writing the snapshot.
		c.MaxBytesBeforeExternalGroupBy = c.MaxMemoryUsage / 4
	}
}

func (c SnapshotConfig) Validate() error {
	if c.Timeout < time.Second || c.MaxThreads == 0 || c.MaxMemoryUsage == 0 {
		return fmt.Errorf("invalid serving rebuild limits: require timeout >= 1s, max_threads > 0 and max_memory_usage > 0")
	}
	if c.MaxBytesBeforeExternalGroupBy == 0 || c.MaxBytesBeforeExternalGroupBy >= c.MaxMemoryUsage {
		return fmt.Errorf("serving rebuild max_bytes_before_external_group_by must be positive and less than max_memory_usage")
	}
	return nil
}

func (c SnapshotConfig) settings() clickhousego.Settings {
	return clickhousego.Settings{
		"max_threads":                        c.MaxThreads,
		"max_insert_threads":                 uint64(1),
		"max_memory_usage":                   c.MaxMemoryUsage,
		"max_bytes_before_external_group_by": c.MaxBytesBeforeExternalGroupBy,
		"max_execution_time":                 uint64(c.Timeout / time.Second),
		"alter_sync":                         1, // Wait for publication on this replica, not all replicas.
	}
}

type SnapshotStorage struct {
	l       xlog.Logger
	options *clickhousego.Options
}

func NewSnapshotStorage(l xlog.Logger, conf *clickhouse.Config) (*SnapshotStorage, error) {
	options, err := clickhouse.BuildOptions(conf)
	if err != nil {
		return nil, err
	}
	if options.Protocol != clickhousego.Native || len(options.Addr) == 0 {
		return nil, fmt.Errorf("snapshot storage requires native ClickHouse addresses")
	}
	return &SnapshotStorage{l: l.WithName("cluster_top_snapshot"), options: options}, nil
}

// RebuildServingSnapshot must run exclusively under the scheduler leadership
// lease. One native session owns synchronization, temporary staging and publish.
// Never retry INSERT SELECT in-place: a failed insert may have committed parts.
func (s *SnapshotStorage) RebuildServingSnapshot(ctx context.Context, generation uint32, config SnapshotConfig) error {
	config.FillDefault()
	if err := config.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, config.Timeout)
	defer cancel()
	conn, err := s.openSession(ctx, config.Timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	queryID := "cluster-top-serving-" + uuid.NewString()
	s.l.Info(ctx, "Rebuilding serving snapshot", log.UInt32("generation", generation), log.String("query_id", queryID))
	ctx = clickhousego.Context(ctx, clickhousego.WithSettings(config.settings()))
	return rebuildServingSnapshot(ctx, conn, generation, queryID)
}

const buildServingSnapshotSQL = `INSERT INTO %s
    (generation, event_type, sort_type, sort_cycles, function, self_cycles, cumulative_cycles)
SELECT generation, event_type, sort_type,
       if(sort_type = 'self_cycles', self_cycles, cumulative_cycles), function, self_cycles, cumulative_cycles
FROM (
    SELECT generation, event_type, function,
           sum(self_cycles) AS self_cycles, sum(cumulative_cycles) AS cumulative_cycles
    FROM cluster_top_by_function_v3
    WHERE generation = ?
    GROUP BY generation, event_type, function
)
ARRAY JOIN ['self_cycles', 'cumulative_cycles'] AS sort_type`

type snapshotConnection interface {
	Exec(context.Context, string, ...any) error
	QueryRow(context.Context, string, ...any) snapshotRowScanner
}

func rebuildServingSnapshot(ctx context.Context, conn snapshotConnection, generation uint32, queryID string) error {
	// Generate a SQL-safe identifier once and reuse it throughout this rebuild.
	stagingTable := "cluster_top_by_function_v3_staging_" + strings.ReplaceAll(uuid.NewString(), "-", "_")
	// Workers may acknowledge inserts on another replica. Read the replicated
	// source only after its queued parts have been applied on this replica.
	if err := conn.Exec(ctx, "SYSTEM SYNC REPLICA "+clusterTopByFunctionTable); err != nil {
		return fmt.Errorf("synchronize snapshot source: %w", err)
	}
	// Copy the serving schema, keys and indexes, but keep the intermediate parts
	// local to this session. Closing it removes staging even after a failed build.
	if err := conn.Exec(ctx, "CREATE TEMPORARY TABLE "+stagingTable+" AS "+clusterTopServingTable+" ENGINE = MergeTree"); err != nil {
		return fmt.Errorf("create temporary staging snapshot: %w", err)
	}
	buildCtx := clickhousego.Context(ctx, clickhousego.WithQueryID(queryID))
	if err := conn.Exec(buildCtx, fmt.Sprintf(buildServingSnapshotSQL, stagingTable), generation); err != nil {
		return fmt.Errorf("build serving snapshot: %w", err)
	}
	var selfRows, cumulativeRows, invalidRows uint64
	if err := conn.QueryRow(ctx, `SELECT countIf(sort_type = 'self_cycles'), countIf(sort_type = 'cumulative_cycles'),
        countIf(sort_type NOT IN ('self_cycles', 'cumulative_cycles') OR sort_cycles != if(sort_type = 'self_cycles', self_cycles, cumulative_cycles))
        FROM `+stagingTable+` WHERE generation = ?`, generation).
		Scan(&selfRows, &cumulativeRows, &invalidRows); err != nil {
		return fmt.Errorf("validate serving snapshot: %w", err)
	}
	if selfRows != cumulativeRows || invalidRows != 0 {
		return fmt.Errorf("invalid serving snapshot: self=%d cumulative=%d invalid=%d", selfRows, cumulativeRows, invalidRows)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	// Empty generations have no partition to replace. Removing an old partition
	// publishes the empty result; DROP on an absent partition is a no-op.
	publishSQL := "ALTER TABLE " + clusterTopServingTable + " DROP PARTITION ?"
	if selfRows != 0 {
		publishSQL = "ALTER TABLE " + clusterTopServingTable + " REPLACE PARTITION ? FROM " + stagingTable
	}
	if err := conn.Exec(ctx, publishSQL, generation); err != nil {
		return fmt.Errorf("publish serving snapshot: %w", err)
	}
	return nil
}
