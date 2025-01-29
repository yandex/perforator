package testutils

import (
	"context"

	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/perforator/pkg/postgres"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func NewTestCluster(pingCtx context.Context, l xlog.Logger) (*hasql.Cluster, error) {
	cfg, err := DefaultTestConfig()
	if err != nil {
		return nil, err
	}

	return postgres.NewCluster(pingCtx, l, &cfg)
}
