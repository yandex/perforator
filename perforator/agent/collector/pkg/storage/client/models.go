package client

import (
	"context"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/binary"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

type BinaryStorage interface {
	StoreBinary(ctx context.Context, buildID string, attributes *perforatorstorage.BinaryAttributes, binary binary.SealedFile) error
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
