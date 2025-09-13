package server

import (
	"context"
	"errors"
	"reflect"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/perforator/internal/symbolizer/auth"
	"github.com/yandex/perforator/perforator/proto/perforator"
)

type acl interface {
	check(ctx context.Context, req any, info *grpc.UnaryServerInfo) error
}

type nopAccessChecker struct{}

func (*nopAccessChecker) check(ctx context.Context, req any, info *grpc.UnaryServerInfo) error {
	return nil
}

type whitelistAccessChecker struct {
	allowedUsers map[string]struct{}
}

func newWhitelistAccessChecker(users []string) acl {
	checker := &whitelistAccessChecker{
		allowedUsers: make(map[string]struct{}),
	}

	for _, user := range users {
		checker.allowedUsers[user] = struct{}{}
	}

	return checker
}

func (c *whitelistAccessChecker) check(ctx context.Context, req any, info *grpc.UnaryServerInfo) error {
	user := auth.UserFromContext(ctx)
	if user == nil {
		return status.Error(codes.Unauthenticated, ErrUserUnspecified.Error())
	}

	_, ok := c.allowedUsers[user.Login]
	if !ok {
		return status.Errorf(codes.PermissionDenied, "permission denied for user %s", user.Login)
	}

	return nil
}

type methodAccessChecker struct {
	acl     acl
	methods map[string]struct{}
}

func newMethodAccessChecker(acl acl, methods map[string]struct{}) acl {
	return &methodAccessChecker{
		acl:     acl,
		methods: methods,
	}
}

func (c *methodAccessChecker) check(ctx context.Context, req any, info *grpc.UnaryServerInfo) error {
	_, ok := c.methods[info.FullMethod]
	if !ok {
		return nil
	}

	return c.acl.check(ctx, req, info)
}

type taskAccessChecker struct {
	acl          acl
	taskSpecType reflect.Type
}

func newTaskAccessChecker(acl acl, taskSpecType reflect.Type) acl {
	return &taskAccessChecker{
		acl:          acl,
		taskSpecType: taskSpecType,
	}
}

func (c *taskAccessChecker) check(ctx context.Context, req any, info *grpc.UnaryServerInfo) error {
	startTaskReq, ok := req.(*perforator.StartTaskRequest)
	if !ok {
		return nil
	}

	taskSpecKind := startTaskReq.GetSpec().GetKind()
	if taskSpecKind == nil {
		return errors.New("unspecified task spec")
	}

	if reflect.TypeOf(taskSpecKind) != c.taskSpecType {
		return nil
	}

	return c.acl.check(ctx, req, info)
}

func newAccessUnaryInterceptor(conf ACLConfig) grpc.UnaryServerInterceptor {
	var accessChecker acl = &nopAccessChecker{}
	if len(conf.RecordRemoteUsers) > 0 {
		accessChecker = newMethodAccessChecker(
			newTaskAccessChecker(
				newWhitelistAccessChecker(conf.RecordRemoteUsers),
				reflect.TypeOf(&perforator.TaskSpec_RecordRemoteProfile{}),
			),
			map[string]struct{}{
				"/NPerforator.NProto.TaskService/StartTask": {},
			},
		)
	}

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		err := accessChecker.check(ctx, req, info)
		if err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func newAccessStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		return handler(srv, ss)
	}
}
