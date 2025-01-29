package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.yandex/hasql/checkers"
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func NewCluster(pingCtx context.Context, l xlog.Logger, conf *Config) (*hasql.Cluster, error) {
	nodes := make([]hasql.Node, 0, len(conf.Endpoints))
	for _, endpoint := range conf.Endpoints {
		connectionString, err := ConnectionString(&conf.AuthConfig, conf.DB, &endpoint, conf.SSLMode, conf.SSLRootCert)
		if err != nil {
			return nil, fmt.Errorf("failed to create connection string for postgres %v: %w", endpoint, err)
		}

		db, err := sqlx.Open("pgx", connectionString)
		if err != nil {
			return nil, fmt.Errorf("failed to open postgres %v: %w", endpoint, err)
		}

		err = db.PingContext(pingCtx)
		if err != nil {
			l.Error(pingCtx, "Failed to ping postgres on start", log.Any("endpoint", endpoint), log.Error(err))
		}

		nodes = append(nodes, hasql.NewNode(endpoint.Addr(), db))
	}

	cluster, err := hasql.NewCluster(
		nodes,
		checkers.PostgreSQL,
		hasql.WithNodePicker(hasql.PickNodeRoundRobin()),
		hasql.WithUpdateInterval(2*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	return cluster, nil
}
