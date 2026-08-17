package combined

import (
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	clustertop "github.com/yandex/perforator/perforator/pkg/storage/cluster_top"
	clickhousestore "github.com/yandex/perforator/perforator/pkg/storage/cluster_top/aggregated"
	postgres "github.com/yandex/perforator/perforator/pkg/storage/cluster_top/generations"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type CombinedClusterTopStorage struct {
	l xlog.Logger
	clickhousestore.AggregationStorage
	postgres.GenerationsStorage
}

func NewCombinedClusterTopStorage(l xlog.Logger, c *clickhouse.Connection, h *hasql.Cluster, asyncInsertConfig clustertop.AsyncInsertConfig) *CombinedClusterTopStorage {
	l = l.WithName("cluster-top-store")

	return &CombinedClusterTopStorage{
		l:                  l,
		AggregationStorage: clickhousestore.NewStorage(l, c, asyncInsertConfig),
		GenerationsStorage: postgres.NewStorage(l, h),
	}
}
