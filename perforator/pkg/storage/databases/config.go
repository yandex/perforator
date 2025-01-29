package databases

import (
	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/postgres"
	"github.com/yandex/perforator/perforator/pkg/s3"
)

type Config struct {
	PostgresCluster  *postgres.Config   `yaml:"postgres"`
	S3Config         *s3.Config         `yaml:"s3"`
	ClickhouseConfig *clickhouse.Config `yaml:"clickhouse"`
}
