package symbolize

import (
	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/pkg/xelf"
)

type BinaryPathProvider interface {
	Path(mapping *pprof.Mapping) string
}

type fixedBinariesPathProvider struct {
	binaryPathByBuildID map[string]string
}

func (p *fixedBinariesPathProvider) Path(mapping *pprof.Mapping) string {
	if path, ok := p.binaryPathByBuildID[mapping.BuildID]; ok {
		return path
	}
	return ""
}

func NewFixedBinariesPathProvider(binaryPaths []string) (BinaryPathProvider, error) {
	binaryPathByBuildID := make(map[string]string)
	for _, binaryPath := range binaryPaths {
		buildID, err := xelf.GetBuildID(binaryPath)
		if err != nil {
			return nil, err
		}
		binaryPathByBuildID[buildID] = binaryPath
	}
	return &fixedBinariesPathProvider{
		binaryPathByBuildID: binaryPathByBuildID,
	}, nil
}

type nilPathProvider struct{}

func (*nilPathProvider) Path(*pprof.Mapping) string {
	return ""
}

func NewNilPathProvider() BinaryPathProvider {
	return &nilPathProvider{}
}

var _ BinaryPathProvider = (*nilPathProvider)(nil)
var _ BinaryPathProvider = (*fixedBinariesPathProvider)(nil)
