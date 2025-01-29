package blob

import (
	"github.com/aws/aws-sdk-go/service/s3"
)

type options struct {
	fsPath   string
	s3bucket string
	s3client *s3.S3
}

func defaultOpts() *options {
	return &options{}
}

type Option = func(o *options)

func WithS3(client *s3.S3, bucket string) Option {
	return func(o *options) {
		o.s3bucket = bucket
		o.s3client = client
	}
}

func WithFS(rootPath string) Option {
	return func(o *options) {
		o.fsPath = rootPath
	}
}
