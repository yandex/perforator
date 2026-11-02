package server

import (
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/services"
	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/services/cluster_top"
	"github.com/yandex/perforator/perforator/internal/symbolizer/proxy/services/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func newServices(
	features *FeaturesConfig,
	clusterTopConfig cluster_top.Config,
	l xlog.Logger,
	reg metrics.Registry,
	storageBundle *bundle.StorageBundle,
) (res []services.GRPCService, err error) {
	if features.EnableCPOExperimental != nil && *features.EnableCPOExperimental {
		res = append(res, custom_profiling_operation.NewService(l, reg, storageBundle.CustomProfilingOperationStorage))
	}

	if features.EnableClusterTopService != nil && *features.EnableClusterTopService {
		res = append(res, cluster_top.NewService(l, storageBundle.ClusterTopGenerationsStorage, clusterTopConfig))
	}

	return res, nil
}
