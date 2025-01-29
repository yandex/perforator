package endpointsetresolver

import (
	"fmt"

	"google.golang.org/grpc"

	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func GetGrpcTargetAndResolverOpts(conf EndpointSetConfig, l xlog.Logger) (string, []grpc.DialOption, error) {
	return "", nil, fmt.Errorf("endpoint set not supported; please, use Host option")
}
