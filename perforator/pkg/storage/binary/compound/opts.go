package binarycompound

import (
	"github.com/aws/aws-sdk-go/service/s3"
	hasql "golang.yandex/hasql/sqlx"
)

type Option = func(*options)

type options struct {
	postgresCluster *hasql.Cluster

	s3client     *s3.S3
	s3bucket     string
	s3GSYMbucket string
}

func defaultOpts() *options {
	return &options{}
}

func WithPostgresMetaStorage(cluster *hasql.Cluster) Option {
	return func(o *options) {
		o.postgresCluster = cluster
	}
}

func WithS3(client *s3.S3, bucket string, gsymBucket string) Option {
	return func(o *options) {
		o.s3client = client
		o.s3bucket = bucket
		o.s3GSYMbucket = gsymBucket
	}
}
