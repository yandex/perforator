package cluster_top

import (
	"context"
	"math/big"

	cluster_top_storage "github.com/yandex/perforator/perforator/pkg/storage/cluster_top"
)

type clickhouseRow struct {
	Generation       int     `ch:"generation"`
	Service          string  `ch:"service"`
	Function         string  `ch:"function"`
	SelfCycles       big.Int `ch:"self_cycles"`
	CumulativeCycles big.Int `ch:"cumulative_cycles"`
}

type ClickhousePerfTopAggregator struct {
	aggregatedStorage cluster_top_storage.Storage
}

const kMaxFunctionNameLength = 512

func (a *ClickhousePerfTopAggregator) Save(ctx context.Context, result *JobResult) error {
	return a.aggregatedStorage.SaveClusterTopEntry(ctx, result)
}

func (a *ClickhousePerfTopAggregator) Print(context.Context) error {
	return nil
}

func NewClickhousePerfTopAggregator(aggregatedStorage cluster_top_storage.Storage) *ClickhousePerfTopAggregator {
	return &ClickhousePerfTopAggregator{
		aggregatedStorage: aggregatedStorage,
	}
}
