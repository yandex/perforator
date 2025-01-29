package tvm

import (
	"fmt"

	"github.com/yandex/perforator/perforator/pkg/storage/creds"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func NewTVMServerInterceptor(
	_ uint32,
	_ string,
	logger xlog.Logger,
) (creds.ServerInterceptor, error) {
	return nil, fmt.Errorf("not supported")
}
