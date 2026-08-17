package factory

import (
	"errors"

	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	clustertop "github.com/yandex/perforator/perforator/pkg/storage/cluster_top"
	"github.com/yandex/perforator/perforator/pkg/storage/cluster_top/combined"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type options struct {
	postgresCluster      *hasql.Cluster
	clickhouseConnection *clickhouse.Connection
	asyncInsertConfig    clustertop.AsyncInsertConfig
}

type Option = func(*options)

func defaultOpts() *options {
	return &options{}
}

func WithPostgresCluster(cluster *hasql.Cluster) Option {
	return func(o *options) {
		o.postgresCluster = cluster
	}
}

func WithClickhouseConnection(connection *clickhouse.Connection) Option {
	return func(o *options) {
		o.clickhouseConnection = connection
	}
}

func WithAsyncInsertConfig(config clustertop.AsyncInsertConfig) Option {
	return func(o *options) {
		o.asyncInsertConfig = config
	}
}

func NewStorage(
	logger xlog.Logger,
	optAppliers ...Option,
) (clustertop.Storage, error) {
	options := defaultOpts()
	for _, optApplier := range optAppliers {
		optApplier(options)
	}

	if options.postgresCluster == nil {
		return nil, errors.New("postgres cluster is not specified")
	}

	if options.clickhouseConnection == nil {
		return nil, errors.New("clickhouse connection is not specified")
	}

	return combined.NewCombinedClusterTopStorage(logger, options.clickhouseConnection, options.postgresCluster, options.asyncInsertConfig), nil

}
