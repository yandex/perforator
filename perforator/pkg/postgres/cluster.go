package postgres

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.yandex/hasql/checkers"
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const maxApplicationNameLength = 63

func NewCluster(ctx context.Context, pingCtx context.Context, l xlog.Logger, app string, conf *Config) (*hasql.Cluster, error) {
	appName := conf.ApplicationNameOverride
	if appName == "" {
		hostname, err := os.Hostname()
		if err != nil {
			l.Warn(ctx, "Failed to enrich application_name with hostname", log.Error(err))
			hostname = fmt.Sprintf("<unresolved-%d>", os.Getpid())
		}
		appName = fmt.Sprintf("perforator-%s@%s", app, hostname)
		if len(appName) > maxApplicationNameLength {
			appName = appName[:maxApplicationNameLength]
		}
	}
	nodes := make([]hasql.Node, 0, len(conf.Endpoints))
	for _, endpoint := range conf.Endpoints {
		connectionString, err := ConnectionString(&conf.AuthConfig, conf.DB, &endpoint, conf.SSLMode, conf.SSLRootCert, appName)
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
