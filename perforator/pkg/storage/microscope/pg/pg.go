package pg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	hasql "golang.yandex/hasql/sqlx"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/perforator/pkg/postgres"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/sqlbuilder"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	AllColumns = "id, user_id, selector, from_ts, to_ts, created_at"
)

var (
	InfinityTS = time.Unix(1<<50-1, 0)
)

type Storage struct {
	l       xlog.Logger
	c       *postgres.Config
	cluster *hasql.Cluster
}

func NewPostgresMicroscopeStorage(l xlog.Logger, cluster *hasql.Cluster) *Storage {
	return &Storage{
		l:       l.WithName("PostgresMicroscopeStorage"),
		cluster: cluster,
	}
}

func (s *Storage) AddMicroscope(ctx context.Context, userID string, selector *querylang.Selector) (*uuid.UUID, error) {
	interval, err := profilequerylang.ParseTimeInterval(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse time interval from selector: %w", err)
	}

	if interval.From == nil || interval.To == nil {
		return nil, errors.New("both timestamp bounds must be specified (`from` timestamp and `to` timestamp)")
	}

	selectorStr, err := profilequerylang.SelectorToString(selector)
	if err != nil {
		return nil, fmt.Errorf("failed to convert selector to string: %w", err)
	}

	uid, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("failed to generate uuidV7: %w", err)
	}

	primary, err := s.cluster.WaitForPrimary(ctx)
	if err != nil {
		return nil, err
	}

	_, err = primary.DBx().ExecContext(
		ctx,
		"INSERT INTO microscopes(id, user_id, selector, from_ts, to_ts) VALUES ($1, $2, $3, $4, $5)",
		uid.String(),
		userID,
		selectorStr,
		*interval.From,
		*interval.To,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert microscope %s of user %s: %w", selectorStr, userID, err)
	}

	s.l.Info(ctx,
		"Inserted microscope",
		log.String("selector", selectorStr),
		log.String("user", userID),
		log.String("uuid", uid.String()),
		log.Time("from", *interval.From),
		log.Time("to", *interval.To),
	)

	return &uid, nil
}

func (s *Storage) ListMicroscopes(
	ctx context.Context,
	filters *microscope.Filters,
	pagination *util.Pagination,
) ([]microscope.Microscope, error) {
	builder := sqlbuilder.Select().
		From("microscopes").
		Values(AllColumns).
		Offset(pagination.Offset).
		OrderBy(&sqlbuilder.OrderBy{Columns: []string{"from_ts", "to_ts"}})

	if pagination.Limit != 0 {
		builder.Limit(pagination.Limit)
	}

	substitutionArgs := make([]interface{}, 0, 5)
	if filters.StartsAfter != nil {
		builder.Where(fmt.Sprintf("from_ts >= $%d", len(substitutionArgs)+1))
		substitutionArgs = append(substitutionArgs, *filters.StartsAfter)
	}
	if filters.StartsBefore != nil {
		builder.Where(fmt.Sprintf("from_ts <= $%d", len(substitutionArgs)+1))
		substitutionArgs = append(substitutionArgs, *filters.StartsBefore)
	}
	if filters.EndsAfter != nil {
		builder.Where(fmt.Sprintf("to_ts >= $%d", len(substitutionArgs)+1))
		substitutionArgs = append(substitutionArgs, *filters.EndsAfter)
	}
	if filters.EndsBefore != nil {
		builder.Where(fmt.Sprintf("to_ts <= $%d", len(substitutionArgs)+1))
		substitutionArgs = append(substitutionArgs, *filters.EndsBefore)
	}

	if filters.User != "" && filters.User != microscope.AllUsers {
		builder.Where(fmt.Sprintf("user_id = $%d", len(substitutionArgs)+1))
		substitutionArgs = append(substitutionArgs, filters.User)
	}

	rows := []microscope.Microscope{}
	sql, err := builder.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, err
	}

	err = alive.DBx().SelectContext(
		ctx,
		&rows,
		sql,
		substitutionArgs...,
	)
	if err != nil {
		return nil, fmt.Errorf("failed select: %w", err)
	}

	s.l.Info(ctx, "Done list microscopes", log.String("sql", sql))

	return rows, nil
}

func (s *Storage) GetUserInfo(ctx context.Context, userID string, opts *microscope.GetUserInfoOptions) (userInfo *microscope.UserInfo, err error) {
	userInfo = &microscope.UserInfo{}

	tsFrom := time.Time{}
	if opts.MicroscopeCountWindow != time.Duration(0) {
		tsFrom = time.Now().Add(-opts.MicroscopeCountWindow)
	}

	alive, err := s.cluster.WaitForAlive(ctx)
	if err != nil {
		return nil, err
	}

	err = alive.DBx().GetContext(
		ctx,
		&userInfo.Microscopes,
		"SELECT COUNT(*) FROM microscopes WHERE user_id = $1 AND created_at BETWEEN $2 AND NOW()",
		userID,
		tsFrom,
	)
	return
}
