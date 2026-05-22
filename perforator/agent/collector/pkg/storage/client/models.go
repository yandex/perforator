package client

import (
	"context"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
)

type BinaryStorage interface {
	StoreBinary(ctx context.Context, buildID string, binary binary.SealedFile) error
	AnnounceBinaries(ctx context.Context, buildIDs []string) ([]string, error)
}

type LabeledProfile struct {
	Profile *profile.Profile
	Labels  map[string]string
}

type ProfileStorage interface {
	StoreProfile(ctx context.Context, profile LabeledProfile) error
}

type Storage interface {
	BinaryStorage
	ProfileStorage
}
