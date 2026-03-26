package binarycompound

import (
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/perforator/pkg/s3"
)

type Option = func(*options)

type options struct {
	postgresCluster *hasql.Cluster

	s3client *s3.Client
	s3bucket string
}

func defaultOpts() *options {
	return &options{}
}

func WithPostgresMetaStorage(cluster *hasql.Cluster) Option {
	return func(o *options) {
		o.postgresCluster = cluster
	}
}

func WithS3(client *s3.Client, bucket string) Option {
	return func(o *options) {
		o.s3client = client
		o.s3bucket = bucket
	}
}
