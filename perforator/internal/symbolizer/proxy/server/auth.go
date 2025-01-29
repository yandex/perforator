package server

import (
	"github.com/yandex/perforator/perforator/internal/symbolizer/auth"
	"github.com/yandex/perforator/perforator/internal/symbolizer/auth/nopauth"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func newAuthProvider(logger xlog.Logger, insecure bool) (auth.Provider, error) {
	return nopauth.NewProvider(), nil
}
