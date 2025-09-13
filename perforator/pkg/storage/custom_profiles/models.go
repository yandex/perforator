package custom_profiles

import (
	"context"

	"github.com/yandex/perforator/perforator/pkg/storage/custom_profiles/meta"
	"github.com/yandex/perforator/perforator/proto/profile"
)

type Storage interface {
	StoreCustomProfile(
		ctx context.Context,
		meta *meta.CustomProfileMeta,
		profile *profile.ProfileContainer,
	) (profileID string, err error)

	GetOperationProfiles(
		ctx context.Context,
		operationID string,
	) ([]*meta.CustomProfileMeta, error)

	FetchProfile(
		ctx context.Context,
		meta *meta.CustomProfileMeta,
	) (*profile.ProfileContainer, error)
}
