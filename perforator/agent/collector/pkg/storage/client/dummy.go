package client

import (
	"context"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
)

////////////////////////////////////////////////////////////////////////////////

var _ Storage = (*DummyStorage)(nil)

////////////////////////////////////////////////////////////////////////////////

type DummyStorage struct{}

func (s *DummyStorage) StoreProfile(ctx context.Context, _ LabeledProfile) error {
	return nil
}

func (s *DummyStorage) StoreBinary(ctx context.Context, buildID string, binary binary.SealedFile) error {
	return nil
}

func (s *DummyStorage) HasBinary(ctx context.Context, buildID string) (bool, error) {
	return true, nil
}

func (s *DummyStorage) AnnounceBinaries(ctx context.Context, buildIDs []string) ([]string, error) {
	return nil, nil
}

////////////////////////////////////////////////////////////////////////////////
