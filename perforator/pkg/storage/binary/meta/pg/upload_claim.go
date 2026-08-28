package pg

import (
	"context"

	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

type uploadClaim struct {
	l           xlog.Logger
	cluster     *hasql.Cluster
	buildID     string
	compression compressionpb.CompressionMethod
}

func (c *uploadClaim) Commit(ctx context.Context, blobInfo *storage.BlobInfo) error {
	primary, err := c.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}

	if c.compression == compressionpb.CompressionMethod_None {
		_, err = primary.DBx().ExecContext(
			ctx,
			`UPDATE binaries
				SET last_used_timestamp = NOW(),
					blob_size = $1,
					uncompressed_size = $2,
					upload_status = $3
				WHERE build_id = $4`,
			blobInfo.Size,
			blobInfo.Size,
			binarymeta.Uploaded,
			c.buildID,
		)
	} else {
		var compressionStr string
		compressionStr, err = compressionMethodToString(ctx, c.compression)
		if err != nil {
			return err
		}
		_, err = primary.DBx().ExecContext(
			ctx,
			`UPDATE binaries
				SET last_used_timestamp = NOW(),
					upload_status = $1,
					compression = $2,
					blob_size = $3
				WHERE build_id = $4`,
			binarymeta.Uploaded,
			compressionStr,
			blobInfo.Size,
			c.buildID,
		)
	}
	if err != nil {
		return err
	}

	c.l.Info(ctx, "Committed binary", log.String("build_id", c.buildID))

	return nil
}

func (c *uploadClaim) Ping(ctx context.Context) error {
	primary, err := c.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}

	_, err = primary.DBx().ExecContext(
		ctx,
		`UPDATE binaries
			SET last_used_timestamp = NOW()
			WHERE build_id = $1`,
		c.buildID,
	)
	if err != nil {
		return err
	}

	return nil
}

func (c *uploadClaim) Abort(ctx context.Context) error {
	primary, err := c.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}

	_, err = primary.DBx().ExecContext(
		ctx,
		`DELETE FROM binaries
			WHERE build_id = $1`,
		c.buildID,
	)
	if err != nil {
		return err
	}

	c.l.Info(ctx, "Aborted binary upload", log.String("build_id", c.buildID))
	return nil
}
