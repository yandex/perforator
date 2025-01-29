package profile

import (
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta/clickhouse"
)

type Config struct {
	MetaStorage             clickhouse.Config `yaml:"meta"`
	S3Bucket                string            `yaml:"bucket"`
	BlobDownloadConcurrency uint32            `yaml:"blob_download_concurrency"`
}
