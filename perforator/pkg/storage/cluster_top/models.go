package clustertop

import (
	"context"
	"math/big"

	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/aggregated"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type GenerationsStorageType string

const (
	Postgres GenerationsStorageType = "postgres"
)

type AggregationStorageType string

const (
	Clickhouse AggregationStorageType = "clickhouse"
)

type Config struct {
	GenerationsStorage GenerationsStorageType `yaml:"generations_storage"`
	AggregationStorage AggregationStorageType `yaml:"aggregation_storage"`
}

type Storage interface {
	ListGenerations(ctx context.Context) ([]*perforator.ClusterTopGeneration, error)
	AggregateClusterTop(ctx context.Context, generation uint32, filter *aggregated.Filter, aggregationType aggregated.GroupByMode, pagination util.Pagination, sortOrder aggregated.SortOrder) ([]*aggregated.AggregationValue, error)
	SaveClusterTopEntry(ctx context.Context, servicePerfTop *aggregated.ServicePerfTop) error
	CountTotalCumulativeCycles(ctx context.Context, generation uint32, totalFunctionName string) (*big.Int, error)
	CountTotalSelfCycles(ctx context.Context, generation uint32, options ...aggregated.CountTotalSelfCyclesOption) (*big.Int, error)
}
