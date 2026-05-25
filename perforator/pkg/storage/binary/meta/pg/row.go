package pg

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

const (
	AllColumns = "build_id, blob_size, ts, attributes, upload_status, last_used_timestamp, compression, uncompressed_size"
)

type BinaryRow struct {
	BuildID                    string    `db:"build_id"`
	UnverifiedUncompressedSize uint64    `db:"blob_size"`
	GSYMBlobSize               uint64    `db:"gsym_blob_size"`
	Timestamp                  time.Time `db:"ts"`
	Attributes                 []byte    `db:"attributes"`
	UploadStatus               string    `db:"upload_status"`
	LastUsedTimestamp          time.Time `db:"last_used_timestamp"`
	Compression                string    `db:"compression"`
	UncompressedSize           uint64    `db:"uncompressed_size"`
}

func compressionMethodToString(ctx context.Context, c compressionpb.CompressionMethod) (string, error) {
	switch c {
	case compressionpb.CompressionMethod_None:
		return "", nil
	case compressionpb.CompressionMethod_Zstd:
		return "zstd", nil
	case compressionpb.CompressionMethod_Gzip:
		return "gzip", nil
	default:
		return "", fmt.Errorf("unknown compression method: %s", c.String())
	}
}

func compressionMethodFromString(s string) (compressionpb.CompressionMethod, error) {
	switch s {
	case "zstd":
		return compressionpb.CompressionMethod_Zstd, nil
	case "gzip":
		return compressionpb.CompressionMethod_Gzip, nil
	case "":
		return compressionpb.CompressionMethod_None, nil
	default:
		return compressionpb.CompressionMethod_Unknown, fmt.Errorf("failed to convert compression method: %s", s)
	}
}

func RowToBinaryMeta(row *BinaryRow) (*binarymeta.BinaryMeta, error) {
	compression, err := compressionMethodFromString(row.Compression)
	if err != nil {
		return nil, err
	}

	uncompressedSize := row.UncompressedSize
	if row.UncompressedSize == 0 && row.Compression == "" {
		uncompressedSize = row.UnverifiedUncompressedSize
	}

	res := &binarymeta.BinaryMeta{
		BuildID:           row.BuildID,
		Timestamp:         row.Timestamp,
		LastUsedTimestamp: row.LastUsedTimestamp,
		Status:            binarymeta.UploadStatus(row.UploadStatus),
		Attributes:        make(map[string]string),
		Compression:       compression,
		UncompressedSize:  uncompressedSize,
	}

	if row.UploadStatus == string(binarymeta.Uploaded) {
		res.BlobInfo = &storage.BlobInfo{
			ID:   row.BuildID,
			Size: row.UnverifiedUncompressedSize,
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

	return res, nil
}

func BinaryMetaToRow(ctx context.Context, meta *binarymeta.BinaryMeta) (*BinaryRow, error) {
	compression, err := compressionMethodToString(ctx, meta.Compression)
	if err != nil {
		return nil, fmt.Errorf("failed to convert compression method: %w", err)
	}

	row := &BinaryRow{
		BuildID:           meta.BuildID,
		Timestamp:         meta.Timestamp,
		LastUsedTimestamp: meta.LastUsedTimestamp,
		UploadStatus:      string(meta.Status),
		Compression:       compression,
		UncompressedSize:  meta.UncompressedSize,
	}

	if meta.BlobInfo != nil {
		row.UnverifiedUncompressedSize = meta.BlobInfo.Size
	}

	if len(meta.Attributes) > 0 {
		attributes, err := json.Marshal(meta.Attributes)
		if err == nil {
			row.Attributes = attributes
		}
	}

	return row, nil
}
