package compound

import (
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/aws/aws-sdk-go/service/s3"

	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta/clickhouse"
)

type options struct {
	clickhouseConn          driver.Conn
	clickhouseConf          *clickhouse.Config
	s3client                *s3.S3
	s3bucket                string
	blobDownloadConcurrency uint32
}

func defaultOpts() *options {
	return &options{}
}

type Option = func(o *options)

func WithClickhouseMetaStorage(conn driver.Conn, conf *clickhouse.Config) Option {
	return func(o *options) {
		o.clickhouseConn = conn
		o.clickhouseConf = conf
	}
}

func WithS3(client *s3.S3, bucket string) Option {
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
