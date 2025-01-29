package config

import (
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta/clickhouse"
)

type Config struct {
	Clickhouse *clickhouse.Config `yaml:"clickhouse"`
}
