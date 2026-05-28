package blob

import "github.com/yandex/perforator/perforator/pkg/storage/blob/models"

type Handle struct {
	Storage  models.Storage
	Download models.ParallelDownloadConfig
}
