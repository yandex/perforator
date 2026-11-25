package kubelet

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/cgroups"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/deploysystem"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const snapshotTimeout = time.Minute

type snapshotSource struct {
	lister      *PodsLister
	initialized bool
}

func NewSource(l xlog.Logger, k Config, refreshPeriod time.Duration, cgroupHints *cgroups.CgroupHints) (deploysystem.Source, error) {
	cgroupPrefix, err := cgroups.FindMountPoint(cgroupHints)
	if err != nil {
		return nil, err
	}
	lister, err := NewPodsLister(l, k.TopologyLableKey, KubeletSettingsOverrides{
		CgroupDriver: k.KubeletCgroupDriver, CgroupRoot: k.KubeletCgroupRoot,
		CgroupsQOSMode: k.KubeletCgroupQOSMode, CgroupContainerPrefix: k.KubeletCgroupContainerPrefix,
		KubernetesAPIServerHost: k.KubernetesAPIServerHost,
	}, cgroupPrefix)
	if err != nil {
		return nil, err
	}
	source := &snapshotSource{lister: lister}
	return deploysystem.NewPollingSnapshotAdapter(source, refreshPeriod, snapshotTimeout)
}

func (s *snapshotSource) Snapshot(ctx context.Context) (deploysystem.Snapshot, error) {
	observedAt := time.Now()
	if !s.initialized {
		if err := s.lister.Init(ctx); err != nil {
			return deploysystem.Snapshot{}, err
		}
		s.initialized = true
	}
	pods, err := s.lister.List(ctx)
	if err != nil {
		return deploysystem.Snapshot{}, err
	}
	snapshot := deploysystem.Snapshot{Host: s.lister.GetHost(), ObservedAt: observedAt}
	for _, pod := range pods {
		w := deploysystem.Pod{
			Name:        pod.name,
			ServiceName: pod.ServiceName(),
			CgroupPath:  deploysystem.RelativeCgroupPath(strings.TrimPrefix(pod.CgroupName(), "/")),
			Metadata:    pod.ProfileLabels(),
		}
		for _, c := range pod.Containers() {
			w.Containers = append(w.Containers, deploysystem.Container{CgroupBaseName: c.CgroupBaseName(), Name: c.Name()})
		}
		snapshot.Pods = append(snapshot.Pods, w)
	}
	return snapshot, nil
}

var invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

func (p *Pod) ProfileLabels() map[string]string {
	labels := map[string]string{
		"namespace": p.namespace,
		"pod":       p.name,
		"cluster":   p.Topology(),
		"service":   p.ServiceName(),
		"cgroup":    p.CgroupName(),
	}
	// Prefix Kubernetes labels and replace characters unsupported in Prometheus label names.
	// For example, app.kubernetes.io/name becomes pod_app_kubernetes_io_name.
	for key, value := range p.labels {
		labels["pod_"+invalidLabelCharRE.ReplaceAllString(key, "_")] = value
	}
	return labels
}
