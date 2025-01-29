package microscope

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
)

const (
	AllUsers = "all"
)

type MicroscopeStorageType string

const (
	Postgres MicroscopeStorageType = "postgres"
)

type Microscope struct {
	ID        string    `db:"id"`
	User      string    `db:"user_id"`
	Selector  string    `db:"selector"`
	FromTS    time.Time `db:"from_ts"`
	ToTS      time.Time `db:"to_ts"`
	CreatedAt time.Time `db:"created_at"`
}

type UserInfo struct {
	Microscopes uint64
}

type Filters struct {
	User         string
	StartsAfter  *time.Time
	StartsBefore *time.Time
	EndsAfter    *time.Time
	EndsBefore   *time.Time
}

type GetUserInfoOptions struct {
	MicroscopeCountWindow time.Duration
}

type Storage interface {
	AddMicroscope(ctx context.Context, userID string, selector *querylang.Selector) (*uuid.UUID, error)

	ListMicroscopes(ctx context.Context, filters *Filters, pagination *util.Pagination) ([]Microscope, error)

	GetUserInfo(ctx context.Context, userID string, opts *GetUserInfoOptions) (*UserInfo, error)
}
