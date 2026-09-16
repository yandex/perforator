package generations

import (
	"context"

	"github.com/yandex/perforator/perforator/proto/perforator"
)

const (
	StatusScheduled = "scheduled"
	StatusFinished  = "finished"
	StatusDeleting  = "deleting"
)

type GenerationsStorage interface {
	Exists(ctx context.Context, id uint32) (bool, error)
	ListGenerations(ctx context.Context) ([]*perforator.ClusterTopGeneration, error)
}
