package server

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/perforator/internal/symbolizer/auth"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

var (
	ErrUserUnspecified = errors.New("user is unspecified")
)

func (s *PerforatorServer) ListMicroscopes(ctx context.Context, req *perforator.ListMicroscopesRequest) (*perforator.ListMicroscopesResponse, error) {
	user := req.User

	if user == "" {
		userInfo := auth.UserFromContext(ctx)
		if userInfo == nil || userInfo.Login == "" {
			return nil, ErrUserUnspecified
		}

		user = userInfo.Login
	}

	pagination := &util.Pagination{}
	if req.Paginated != nil {
		pagination.Offset = req.Paginated.Offset
		pagination.Limit = req.Paginated.Limit
	}
	if pagination.Limit == 0 {
		pagination.Limit = 500
	}

	filters := &microscope.Filters{
		User: user,
	}
	if req.StartsAfter != nil {
		filters.StartsAfter = ptr.Time(req.StartsAfter.AsTime())
	}

	microscopes, err := s.microscopeStorage.ListMicroscopes(
		ctx,
		filters,
		pagination,
	)
	if err != nil {
		return nil, err
	}

	res := &perforator.ListMicroscopesResponse{
		Microscopes: make([]*perforator.Microscope, 0, len(microscopes)),
	}

	for _, scope := range microscopes {
		res.Microscopes = append(res.Microscopes, &perforator.Microscope{
			Selector: scope.Selector,
			ID:       scope.ID,
			User:     scope.User,
			Interval: &perforator.TimeInterval{
				From: timestamppb.New(scope.FromTS),
				To:   timestamppb.New(scope.ToTS),
			},
		})
	}

	return res, nil
}

func (s *PerforatorServer) throttleMicroscope(ctx context.Context) error {
	user := auth.UserFromContext(ctx)
	if user == nil || user.Login == "" {
		return ErrUserUnspecified
	}

	userInfo, err := s.microscopeStorage.GetUserInfo(
		ctx,
		user.Login,
		&microscope.GetUserInfoOptions{MicroscopeCountWindow: s.c.MicroscopeConfig.Throttle.LimitWindow},
	)
	if err != nil {
		return fmt.Errorf("failed to get user info for user %s: %w", user.Login, err)
	}

	if s.c.MicroscopeConfig.Throttle.LimitPerUser <= uint32(userInfo.Microscopes) {
		return fmt.Errorf(
			"user %s is throttled, got %d microscopes during %s window (allowed max is %d)",
			user.Login,
			userInfo.Microscopes,
			s.c.MicroscopeConfig.Throttle.LimitWindow.String(),
			s.c.MicroscopeConfig.Throttle.LimitPerUser,
		)
	}

	return nil
}

func (s *PerforatorServer) SetMicroscope(ctx context.Context, req *perforator.SetMicroscopeRequest) (*perforator.SetMicroscopeResponse, error) {
	selector, err := profilequerylang.ParseSelector(req.Selector)
	if err != nil {
		return nil, fmt.Errorf("failed to parse selector %s: %w", req.Selector, err)
	}

	err = s.throttleMicroscope(ctx)
	if err != nil {
		return nil, err
	}

	user := auth.UserFromContext(ctx)
	if user == nil || user.Login == "" {
		return nil, ErrUserUnspecified
	}

	uid, err := s.microscopeStorage.AddMicroscope(ctx, user.Login, selector)
	if err != nil {
		return nil, err
	}

	return &perforator.SetMicroscopeResponse{
		ID: uid.String(),
	}, nil
}
