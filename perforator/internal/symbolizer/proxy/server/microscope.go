package server

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/ptr"
	"github.com/yandex/perforator/observability/lib/querylang"
	"github.com/yandex/perforator/observability/lib/querylang/operator"
	"github.com/yandex/perforator/perforator/internal/symbolizer/auth"
	"github.com/yandex/perforator/perforator/pkg/profilequerylang"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope"
	"github.com/yandex/perforator/perforator/pkg/storage/util"
	"github.com/yandex/perforator/perforator/proto/lib/time_interval"
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
			Interval: &time_interval.TimeInterval{
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

func replaceSelectorTimeInterval(selector *querylang.Selector, interval *time_interval.TimeInterval) *querylang.Selector {
	selector.Matchers = slices.DeleteFunc(selector.Matchers, func(matcher *querylang.Matcher) bool {
		return matcher.Field == profilequerylang.TimestampLabel
	})

	selector.Matchers = append(
		selector.Matchers,
		profilequerylang.BuildMatcher(
			profilequerylang.TimestampLabel,
			querylang.AND,
			querylang.Condition{Operator: operator.GTE},
			[]string{interval.From.AsTime().Format(time.RFC3339Nano)},
		),
	)

	selector.Matchers = append(
		selector.Matchers,
		profilequerylang.BuildMatcher(
			profilequerylang.TimestampLabel,
			querylang.AND,
			querylang.Condition{Operator: operator.LTE},
			[]string{interval.To.AsTime().Format(time.RFC3339Nano)},
		),
	)

	return selector
}

func (s *PerforatorServer) sanitizeMicroscope(ctx context.Context, selector *querylang.Selector) error {
	selectorStr, err := profilequerylang.SelectorToString(selector)
	if err != nil {
		return fmt.Errorf("failed to sanitize microscope: %w", err)
	}

	selectorInterval, err := profilequerylang.ParseTimeInterval(selector)
	if err != nil {
		return fmt.Errorf("failed to parse microscope interval: %w", err)
	}
	if selectorInterval.From == nil || selectorInterval.To == nil {
		return fmt.Errorf("failed to parse microscope interval: both from and to bounds should be specified")
	}
	microscopeDurationMinutes := int(selectorInterval.To.Sub(*selectorInterval.From).Minutes())

	// Here we try to ensure that the amount of profiles the microscope would generate
	// is somewhat sane. It would (presumably) always be true for a per-pod or per-node
	// microscope, but not necessary so for a per-service microscope.
	//
	// To implement this sanity check, we load all the profiles metas matching the selector
	// for the last hour, sum their weights and normalize this sum to microscope duration.

	// Take the last hour, should give an estimate good enough
	const kMinutesInAnHour = 60

	query, err := s.parseProfileQuery(&perforator.ProfileQuery{
		Selector: selectorStr,
	})
	if err != nil {
		return fmt.Errorf("failed to process selector: %w", err)
	}
	// This a copy of original selector, which we've acquired via
	// serializing and subsequent deserializing, so we are free to mutate it
	query.Selector = replaceSelectorTimeInterval(query.Selector, &time_interval.TimeInterval{
		From: timestamppb.New(time.Now().Add(-time.Minute * kMinutesInAnHour)),
		To:   timestamppb.Now(),
	})

	profileMetas, err := s.profileStorage.SelectProfiles(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to estimate microscope profiles volume")
	}

	weightOfProfilesPerHour := 0
	for _, profileMeta := range profileMetas {
		profileWeightStr := profileMeta.Attributes[profilequerylang.WeightLabel]
		profileWeight, err := strconv.Atoi(profileWeightStr)
		if err != nil {
			return fmt.Errorf("failed to parse profile weight when calculating microscope profiles volume: %w", err)
		}

		weightOfProfilesPerHour += profileWeight
	}

	// When a microscope is enabled, the profile's weight is 1, so the total weight and total volume are equal.
	estimatedVolumeOfProfiles := weightOfProfilesPerHour * microscopeDurationMinutes / kMinutesInAnHour

	s.l.Info(
		ctx,
		"Estimated volume of profiles for the microscope",
		log.Int("profiles_volume", estimatedVolumeOfProfiles),
	)

	const kMaxExtimatedMicroscopeProfilesVolume = 10000
	if estimatedVolumeOfProfiles > kMaxExtimatedMicroscopeProfilesVolume {
		return fmt.Errorf(
			"estimated profiles volume for microscope (%d) exceeds the maximum amount allowed (%d)",
			estimatedVolumeOfProfiles,
			kMaxExtimatedMicroscopeProfilesVolume,
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

	err = s.sanitizeMicroscope(ctx, selector)
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
