package clickhouse

import (
	"maps"
	"slices"
	"time"

	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
)

////////////////////////////////////////////////////////////////////////////////

type ProfileRow struct {
	ID            string            `ch:"id"`
	System        string            `ch:"system_name"`
	MainEventType string            `ch:"event_type"`
	AllEventTypes []string          `ch:"event_types"`
	Cluster       string            `ch:"cluster"`
	Service       string            `ch:"service"`
	PodID         string            `ch:"pod_id"`
	NodeID        string            `ch:"node_id"`
	Timestamp     time.Time         `ch:"timestamp"`
	BuildIDs      []string          `ch:"build_ids"`
	Attributes    map[string]string `ch:"attributes"`
	Expired       bool              `ch:"expired"`
	Envs          []string          `ch:"envs"`
}

func profileModelFromMeta(p *meta.ProfileMetadata) *ProfileRow {
	return &ProfileRow{
		ID:            p.BlobID,
		System:        p.System,
		MainEventType: p.MainEventType,
		AllEventTypes: p.AllEventTypes,
		Cluster:       p.Cluster,
		Service:       p.Service,
		PodID:         p.PodID,
		NodeID:        p.NodeID,
		Timestamp:     p.Timestamp,
		BuildIDs:      slices.Clone(p.BuildIDs),
		Attributes:    maps.Clone(p.Attributes),
		Envs:          p.Envs,
	}
}

func profileMetaFromModel(p *ProfileRow) *meta.ProfileMetadata {
	return &meta.ProfileMetadata{
		ID:            p.ID,
		BlobID:        p.ID,
		System:        p.System,
		MainEventType: p.MainEventType,
		AllEventTypes: p.AllEventTypes,
		Cluster:       p.Cluster,
		Service:       p.Service,
		PodID:         p.PodID,
		NodeID:        p.NodeID,
		Timestamp:     p.Timestamp,
		BuildIDs:      slices.Clone(p.BuildIDs),
		Attributes:    maps.Clone(p.Attributes),
		Envs:          p.Envs,
	}
}

////////////////////////////////////////////////////////////////////////////////

type ServiceRow struct {
	Service      string    `ch:"service"`
	MaxTimestamp time.Time `ch:"max_timestamp"`
	ProfileCount uint64    `ch:"profile_count"`
}

func serviceMetaFromModel(s *ServiceRow) *meta.ServiceMetadata {
	return &meta.ServiceMetadata{
		Service:      s.Service,
		LastUpdate:   s.MaxTimestamp,
		ProfileCount: s.ProfileCount,
	}
}

////////////////////////////////////////////////////////////////////////////////
