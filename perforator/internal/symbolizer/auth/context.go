package auth

import (
	"context"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

////////////////////////////////////////////////////////////////////////////////

type User struct {
	Login string
}

////////////////////////////////////////////////////////////////////////////////

func ContextWithUser(ctx context.Context, user *User) context.Context {
	if user == nil {
		return ctx
	}
	ctx = xlog.WrapContext(ctx, log.String("user", user.Login))
	return context.WithValue(ctx, userKey, user)
}

func UserFromContext(ctx context.Context) *User {
	u, _ := ctx.Value(userKey).(*User)
	return u
}

////////////////////////////////////////////////////////////////////////////////

type key struct{}

var userKey key

////////////////////////////////////////////////////////////////////////////////
