package client

import (
	"fmt"

	"github.com/yandex/perforator/library/go/core/buildinfo"
)

func makeUserAgentString() string {
	revision := buildinfo.Info.ArcadiaSourceRevision
	if revision == "" {
		revision = buildinfo.Info.Hash
	}
	if buildinfo.Info.Dirty != "" {
		revision += "~dirty"
	}
	return fmt.Sprintf("github.com/yandex/perforator/perforator/symbolizer/pkg/client@%s@%s", buildinfo.Info.Branch, revision)
}
