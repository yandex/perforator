package databases

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/service/s3"
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/postgres"
	s3client "github.com/yandex/perforator/perforator/pkg/s3"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type Databases struct {
	PostgresCluster *hasql.Cluster

	ClickhouseConn *clickhouse.Connection

	S3Client *s3.S3
}

// bgCtx should be valid for as long as databases are used
func NewDatabases(ctx context.Context, bgCtx context.Context, l xlog.Logger, c *Config, app string, reg metrics.Registry) (*Databases, error) {
	res := &Databases{}
	var err error

	if c.S3Config != nil {
		res.S3Client, err = s3client.NewClient(ctx, bgCtx, l, c.S3Config, reg)
		if err != nil {
			return nil, fmt.Errorf("failed to init s3: %w", err)
		}
	}

	if c.PostgresCluster != nil {
		res.PostgresCluster, err = postgres.NewCluster(ctx, bgCtx, l, app, c.PostgresCluster)
		if err != nil {
			return nil, fmt.Errorf("failed to init postgres cluster: %w", err)
		}
	}

	if c.ClickhouseConfig != nil {
		res.ClickhouseConn, err = clickhouse.Connect(ctx, c.ClickhouseConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to init clickhouse conn: %w", err)
		}
	}

	return res, nil
}
