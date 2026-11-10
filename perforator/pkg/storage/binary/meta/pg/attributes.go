package pg

import (
	"context"
	"fmt"

	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

func (s *Storage) MergeAttributes(ctx context.Context, buildID string, attributes *perforatorstorage.BinaryAttributes) error {
	if attributes == nil {
		return nil
	}
	value, err := marshalAttributes(attributes)
	if err != nil {
		return err
	}

	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return err
	}
	result, err := primary.DBx().ExecContext(ctx, `UPDATE binaries
		SET attributes =
			(CASE WHEN jsonb_typeof(attributes) = 'object' THEN attributes ELSE '{}'::jsonb END) || $2::jsonb
		WHERE build_id = $1 AND upload_status = 'uploaded'`, buildID, string(value))
	if err != nil {
		return fmt.Errorf("update binary attributes: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count updated binary rows: %w", err)
	}
	if rows != 1 {
		return fmt.Errorf("binary %q not found or upload is not complete", buildID)
	}
	return nil
}
