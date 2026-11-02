package generations

import (
	"context"

	"github.com/yandex/perforator/perforator/proto/perforator"
)

const (
	StatusScheduled  = "scheduled"
	StatusFinalizing = "finalizing"
	StatusFinished   = "finished"
	StatusDeleting   = "deleting"
)

type GenerationsStorage interface {
	GetGeneration(ctx context.Context, id uint32) (*perforator.ClusterTopGeneration, error)
	Exists(ctx context.Context, id uint32) (bool, error)
	ListGenerations(ctx context.Context) ([]*perforator.ClusterTopGeneration, error)
}
