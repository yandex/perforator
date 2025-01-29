package pg

import (
	"context"

	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	binarymeta "github.com/yandex/perforator/perforator/pkg/storage/binary/meta"
	"github.com/yandex/perforator/perforator/pkg/storage/storage"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type committer struct {
	l       xlog.Logger
	cluster *hasql.Cluster
	buildID string
}

func (c *committer) Commit(ctx context.Context, blobInfo *storage.BlobInfo) error {
	primary, err := c.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}

	_, err = primary.DBx().ExecContext(
		ctx,
		`UPDATE binaries
			SET last_used_timestamp = NOW(),
				blob_size = $1,
				upload_status = $2
			WHERE build_id = $3`,
		blobInfo.Size,
		binarymeta.Uploaded,
		c.buildID,
	)
	if err != nil {
		return err
	}

	c.l.Info(ctx, "Commited binary", log.String("build_id", c.buildID))

	return nil
}

func (c *committer) Ping(ctx context.Context) error {
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

func (c *committer) Abort(ctx context.Context) error {
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
