package certs

import (
	"github.com/yandex/perforator/library/go/core/resource"
)

func InternalCAs() []byte {
	return resource.Get("/certifi/internal.pem")
}

func CommonCAs() []byte {
	return resource.Get("/certifi/common.pem")
}
