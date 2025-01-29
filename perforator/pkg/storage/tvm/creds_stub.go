package tvm

import (
	"fmt"

	"github.com/yandex/perforator/perforator/pkg/storage/creds"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func NewTVMCredentials(
	selfID uint32,
	storageID uint32,
	secret string,
	cacheDir string,
	l xlog.Logger,
) (creds.DestroyablePerRPCCredentials, error) {
	return nil, fmt.Errorf("not supported")
}
