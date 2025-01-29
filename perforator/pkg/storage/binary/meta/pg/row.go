package pg

import (
	"encoding/json"
	"time"

	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
)

const (
	AllColumns = "build_id, blob_size, ts, attributes, upload_status, last_used_timestamp"
)

type BinaryRow struct {
	BuildID           string    `db:"build_id"`
	BlobSize          uint64    `db:"blob_size"`
	GSYMBlobSize      uint64    `db:"gsym_blob_size"`
	Timestamp         time.Time `db:"ts"`
	Attributes        []byte    `db:"attributes"`
	UploadStatus      string    `db:"upload_status"`
	LastUsedTimestamp time.Time `db:"last_used_timestamp"`
}

func RowToBinaryMeta(row *BinaryRow) *binarymeta.BinaryMeta {
	res := &binarymeta.BinaryMeta{
		BuildID:           row.BuildID,
		Timestamp:         row.Timestamp,
		LastUsedTimestamp: row.LastUsedTimestamp,
		Status:            binarymeta.UploadStatus(row.UploadStatus),
		Attributes:        make(map[string]string),
	}

	if row.UploadStatus == string(binarymeta.Uploaded) {
		res.BlobInfo = &storage.BlobInfo{
			ID:   row.BuildID,
			Size: row.BlobSize,
		}

		if row.GSYMBlobSize != 0 {
			res.GSYMBlobInfo = &storage.BlobInfo{
				ID:   row.BuildID,
				Size: row.GSYMBlobSize,
			}
		}
	}

	if len(row.Attributes) > 0 {
		_ = json.Unmarshal(row.Attributes, &res.Attributes)
	}

	return res
}

func BinaryMetaToRow(meta *binarymeta.BinaryMeta) *BinaryRow {
	row := &BinaryRow{
		BuildID:           meta.BuildID,
		Timestamp:         meta.Timestamp,
		LastUsedTimestamp: meta.LastUsedTimestamp,
		UploadStatus:      string(meta.Status),
	}

	if meta.BlobInfo != nil {
		row.BlobSize = meta.BlobInfo.Size
	}

	if len(meta.Attributes) > 0 {
		attributes, err := json.Marshal(meta.Attributes)
		if err == nil {
			row.Attributes = attributes
		}
	}

	return row
}
