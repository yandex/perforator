package offsets

import (
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	"github.com/yandex/perforator/perforator/internal/linguist/common/offsetloader"
)

func decodeVersion(encoded offsetloader.LanguageVersion) *python.PythonVersion {
	return &python.PythonVersion{
		Micro: encoded.Patch(),
		Minor: encoded.Minor(),
		Major: encoded.Major(),
	}
}
