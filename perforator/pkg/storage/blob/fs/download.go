package fs

import "github.com/yandex/perforator/perforator/pkg/storage/blob/models"

func (s *FSStorage) ParallelDownloadConfig() models.ParallelDownloadConfig {
	return models.ParallelDownloadConfig{
		Concurrency: 1,
		PartSize:    1,
	}
}
