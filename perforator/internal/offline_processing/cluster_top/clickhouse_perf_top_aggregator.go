package cluster_top

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

type clickhouseRow struct {
	Generation       int     `ch:"generation"`
	Service          string  `ch:"service"`
	Function         string  `ch:"function"`
	SelfCycles       big.Int `ch:"self_cycles"`
	CumulativeCycles big.Int `ch:"cumulative_cycles"`
}

type ClickhousePerfTopAggregator struct {
	clickhouseConn driver.Conn
}

const kMaxFunctionNameLength = 512

func (a *ClickhousePerfTopAggregator) Save(ctx context.Context, servicePerfTop *ServicePerfTop) error {
	batch, err := a.clickhouseConn.PrepareBatch(
		ctx,
		"INSERT INTO cluster_top(generation, service, function, self_cycles, cumulative_cycles)",
	)
	if err != nil {
		return fmt.Errorf("failed to prepare clickhouse batch: %w", err)
	}

	defer func() { _ = batch.Abort() }()

	for _, function := range servicePerfTop.Functions {
		lengthLimitedFunctionName := function.Name
		if len(lengthLimitedFunctionName) > kMaxFunctionNameLength {
			lengthLimitedFunctionName = lengthLimitedFunctionName[:kMaxFunctionNameLength]
		}
		clickhouseRow := clickhouseRow{
			Generation:       servicePerfTop.Generation,
			Service:          servicePerfTop.ServiceName,
			Function:         lengthLimitedFunctionName,
			SelfCycles:       function.SelfCycles,
			CumulativeCycles: function.CumulativeCycles,
		}
		err := batch.AppendStruct(&clickhouseRow)
		if err != nil {
			return fmt.Errorf("failed to serialize clickhouse row: %w", err)
		}
	}

	return batch.Send()
}

func (a *ClickhousePerfTopAggregator) Print(context.Context) error {
	return nil
}

func NewClickhousePerfTopAggregator(clickhouseConn driver.Conn) *ClickhousePerfTopAggregator {
	return &ClickhousePerfTopAggregator{
		clickhouseConn: clickhouseConn,
	}
}
