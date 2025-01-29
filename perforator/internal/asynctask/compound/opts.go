package compound

import (
	hasql "golang.yandex/hasql/sqlx"

	inmemorytaskservice "github.com/yandex/perforator/perforator/internal/asynctask/inmemory"
	postgrestaskservice "github.com/yandex/perforator/perforator/internal/asynctask/postgres"
)

type Option = func(*options)

type options struct {
	postgresCluster *hasql.Cluster
	postgresConfig  *postgrestaskservice.Config

	inMemoryConfig *inmemorytaskservice.Config
}

func defaultOpts() *options {
	return &options{}
}

func WithPostgresTasksStorage(conf *TasksConfig, cluster *hasql.Cluster) Option {
	return func(o *options) {
		o.postgresCluster = cluster
		o.postgresConfig = &postgrestaskservice.Config{
			PingPeriod:  conf.PingPeriod,
			PingTimeout: conf.PingTimeout,
			MaxAttempts: conf.MaxAttempts,
		}
	}
}

func WithInMemoryTasksStorage(conf *TasksConfig) Option {
	return func(o *options) {
		o.inMemoryConfig = &inmemorytaskservice.Config{
			PingPeriod:  conf.PingPeriod,
			PingTimeout: conf.PingTimeout,
			MaxAttempts: conf.MaxAttempts,
		}
	}
}
