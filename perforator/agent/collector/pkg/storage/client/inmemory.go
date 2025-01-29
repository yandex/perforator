package client

import (
	"context"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
)

////////////////////////////////////////////////////////////////////////////////

var _ Storage = (*InMemoryStorage)(nil)

////////////////////////////////////////////////////////////////////////////////

type InMemoryStorageConfig struct {
	Watermark uint32 `yaml:"watermark"`
}

func (c *InMemoryStorageConfig) FillDefault() {
	if c.Watermark == 0 {
		c.Watermark = 1000
	}
}

////////////////////////////////////////////////////////////////////////////////

type InMemoryStorage struct {
	conf     *InMemoryStorageConfig
	Profiles []LabeledProfile
}

func NewInMemoryStorage(conf *InMemoryStorageConfig) *InMemoryStorage {
	conf.FillDefault()

	return &InMemoryStorage{
		conf:     conf,
		Profiles: make([]LabeledProfile, 0),
	}
}

func (s *InMemoryStorage) StoreProfile(ctx context.Context, profile LabeledProfile) error {
	s.Profiles = append(s.Profiles, profile)
	if len(s.Profiles) > 2*int(s.conf.Watermark) {
		s.Profiles = s.Profiles[s.conf.Watermark:]
	}
	return nil
}

func (s *InMemoryStorage) StoreBinary(ctx context.Context, buildID string, binary binary.SealedFile) error {
	return nil
}

func (s *InMemoryStorage) HasBinary(ctx context.Context, buildID string) (bool, error) {
	return true, nil
}

func (s *InMemoryStorage) AnnounceBinaries(ctx context.Context, buildIDs []string) ([]string, error) {
	return nil, nil
}

////////////////////////////////////////////////////////////////////////////////
