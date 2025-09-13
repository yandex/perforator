package clickhouse

import (
	"maps"
	"slices"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/custom_profiles/meta"
)

////////////////////////////////////////////////////////////////////////////////

type CustomProfileRow struct {
	ID            string            `ch:"id"`
	OperationID   string            `ch:"operation_id"`
	FromTimestamp time.Time         `ch:"from_timestamp"`
	ToTimestamp   time.Time         `ch:"to_timestamp"`
	BuildIDs      []string          `ch:"build_ids"`
	Attributes    map[string]string `ch:"attributes"`
}

func customProfileModelFromMeta(p *meta.CustomProfileMeta) *CustomProfileRow {
	return &CustomProfileRow{
		OperationID:   p.OperationID,
		ID:            p.ID,
		FromTimestamp: p.FromTimestamp,
		ToTimestamp:   p.ToTimestamp,
		BuildIDs:      slices.Clone(p.BuildIDs),
		Attributes:    maps.Clone(p.Attributes),
	}
}

func customProfileMetaFromModel(p *CustomProfileRow) *meta.CustomProfileMeta {
	return &meta.CustomProfileMeta{
		ID:            p.ID,
		OperationID:   p.OperationID,
		FromTimestamp: p.FromTimestamp,
		ToTimestamp:   p.ToTimestamp,
		BuildIDs:      slices.Clone(p.BuildIDs),
		Attributes:    maps.Clone(p.Attributes),
	}
}

////////////////////////////////////////////////////////////////////////////////
