package s3

import (
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/yandex/perforator/perforator/pkg/storage/blob/models"
)

const DownloadConcurrency = 20

func defaultParallelDownloadConfig() models.ParallelDownloadConfig {
	return models.ParallelDownloadConfig{
		Concurrency: DownloadConcurrency,
		PartSize:    s3manager.DefaultDownloadPartSize,
	}
}

func (s *S3Storage) ParallelDownloadConfig() models.ParallelDownloadConfig {
	return models.ParallelDownloadConfig{
		Concurrency: s.downloader.Concurrency,
		PartSize:    s.downloader.PartSize,
	}
}
