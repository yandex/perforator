package cluster_top

import (
	"context"
	"math/big"
	"time"
)

type TimeRange struct {
	From time.Time
	To   time.Time
}

type ServiceProcessingHandler interface {
	GetServiceName() string

	GetGeneration() int

	GetTimeRange() TimeRange

	Finalize(ctx context.Context, processingErr error)
}

type ServiceSelector interface {
	SelectService(ctx context.Context, heavy bool) (ServiceProcessingHandler, error)
}

type Function struct {
	Name             string
	SelfCycles       big.Int
	CumulativeCycles big.Int
}

type ServicePerfTop struct {
	Generation  int
	ServiceName string

	Functions []Function
}

type ClusterPerfTopAggregator interface {
	Save(ctx context.Context, servicePerfTop *ServicePerfTop) error

	Print(ctx context.Context) error
}
