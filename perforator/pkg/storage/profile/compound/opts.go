package compound

import (
	"github.com/yandex/perforator/perforator/pkg/clickhouse"
	"github.com/yandex/perforator/perforator/pkg/s3"
	storage "github.com/yandex/perforator/perforator/pkg/storage/profile"
	clickhouse_meta "github.com/yandex/perforator/perforator/pkg/storage/profile/meta/clickhouse"
)

type options struct {
	clickhouseConn          *clickhouse.Connection
	clickhouseConf          *clickhouse_meta.Config
	s3client                *s3.Client
	s3bucket                string
	blobDownloadConcurrency uint32
	storageOpts             storage.Options
}

func defaultOpts() *options {
	return &options{}
}

type Option = func(o *options)

func WithClickhouseMetaStorage(conn *clickhouse.Connection, conf *clickhouse_meta.Config) Option {
	return func(o *options) {
		o.clickhouseConn = conn
		o.clickhouseConf = conf
	}
}

func WithS3(client *s3.Client, bucket string) Option {
	return func(o *options) {
		o.s3bucket = bucket
		o.s3client = client
	}
}

func WithBlobDownloadConcurrency(concurrency uint32) Option {
	return func(o *options) {
		o.blobDownloadConcurrency = concurrency
	}
}

func WithContainerFormat(enabled bool) Option {
	return func(o *options) {
		o.storageOpts.WriteInContainerFormat = enabled
	}
}
