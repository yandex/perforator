package profiler

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/buildinfo"
	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/deploysystem/kubelet"
	"github.com/yandex/perforator/perforator/pkg/linux/cpuinfo"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	updateCgroupPeriod = 1 * time.Minute
)

type blackListEntryCompiled struct {
	Regexp *regexp.Regexp
	Reason string
}

type containerCgroup struct {
	parentPath string
	baseName   string
}

type containerInfo struct {
	podServiceName string
	containerName  string
}

type PodsCgroupTracker struct {
	l    log.Logger
	conf *config.PodsDeploySystemConfig

	podsLister   *kubelet.PodsLister
	cpuModelName string
	hostname     string

	workloadsmu sync.RWMutex
	workloads   map[containerCgroup]containerInfo

	blackListRegexps []*blackListEntryCompiled
	whiteListRegexps []*regexp.Regexp
}

func (t *PodsCgroupTracker) buildCgroupConfig(
	pod *kubelet.Pod,
) *CgroupConfig {
	labels := t.makeCommonProfileLabels()

	for k, v := range pod.ProfileLabels() {
		labels[k] = v
	}

	// User can explicitly override any gathered label
	for key, val := range t.conf.Labels {
		labels[key] = val
	}

	cgroupConfig := &CgroupConfig{
		Name: pod.CgroupName(),
		Opts: []TraceOption{
			WithProfileLabels(labels),
		},
	}

	return cgroupConfig
}

func (t *PodsCgroupTracker) makeCommonProfileLabels() (labels map[string]string) {
	// There is currently no way to get cluster name from within pod: https://github.com/kubernetes/kubernetes/issues/44954
	labels = map[string]string{
		"host":             t.hostname,
		"cpu":              t.cpuModelName,
		"profiler_version": buildinfo.Info.SVNRevision,
	}

	return
}

// Return reason if black listed.
func (t *PodsCgroupTracker) isPodsetBlackListed(podName string) (bool, string) {
	for _, reg := range t.blackListRegexps {
		if reg.Regexp.MatchString(podName) {
			return true, reg.Reason
		}
	}

	return false, ""
}

func (t *PodsCgroupTracker) isPodsetWhiteListed(podName string) bool {
	for _, reg := range t.whiteListRegexps {
		if reg.MatchString(podName) {
			return true
		}
	}

	return false
}

func (t *PodsCgroupTracker) isPodFiltered(pod *kubelet.Pod) (bool, string) {
	if blisted, reason := t.isPodsetBlackListed(pod.ID()); blisted {
		return true, fmt.Sprintf("podset %s is blacklisted: %s", pod.ID(), reason)
	}

	if t.isPodsetWhiteListed(pod.ID()) {
		return false, ""
	}

	if !t.conf.PodOptions.Default {
		return true, "perforator is disabled by default"
	}

	return false, ""
}

func (t *PodsCgroupTracker) rebuildWorkloadInfo(pods []*kubelet.Pod) {
	workloads := make(map[containerCgroup]containerInfo)
	t.l.Info("Building workload mapping", log.Int("count", len(pods)))
	for _, pod := range pods {
		for _, cont := range pod.Containers() {
			t.l.Info(
				"Adding container",
				log.String("cgroup", cont.CgroupBaseName()),
				log.String("pod", pod.ID()),
				log.String("container", cont.Name()),
			)
			key := containerCgroup{strings.TrimPrefix(pod.CgroupName(), "/"), cont.CgroupBaseName()}
			workloads[key] = containerInfo{
				containerName:  cont.Name(),
				podServiceName: pod.ServiceName(),
			}
		}

	}
	t.l.Info("Saving workload mapping")
	t.workloadsmu.Lock()
	defer t.workloadsmu.Unlock()
	t.workloads = workloads
}

func (t *PodsCgroupTracker) refreshCgroups(ctx context.Context) ([]*CgroupConfig, error) {
	pods, err := t.podsLister.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing pods: %w", err)
	}

	t.l.Info("Found pods", log.Int("count", len(pods)))

	t.rebuildWorkloadInfo(pods)

	cgroups := []*CgroupConfig{}
	for _, pod := range pods {
		filtered, filteringReason := t.isPodFiltered(pod)
		if filtered {
			t.l.Info(
				"Filtered pod",
				log.String("pod_id", pod.ID()),
				log.String("reason", filteringReason),
			)
			continue
		}

		t.l.Info(
			"Found perforator enabled pod",
			log.String("cgroup", pod.CgroupName()),
			log.String("service", pod.ServiceName()),
			log.String("id", pod.ID()),
		)

		cgroups = append(cgroups, t.buildCgroupConfig(pod))
	}

	return cgroups, nil
}

func (t *PodsCgroupTracker) ResolveContainer(parentCgroupPath, containerCgroupBaseName string) (podServiceName, containerName string, found bool) {
	t.workloadsmu.RLock()
	defer t.workloadsmu.RUnlock()
	info, found := t.workloads[containerCgroup{parentCgroupPath, containerCgroupBaseName}]
	return info.podServiceName, info.containerName, found
}

func newPodsCgroupTracker(c *config.PodsDeploySystemConfig, l log.Logger, cgroupPrefix string) (*PodsCgroupTracker, error) {
	l = l.WithName("pod_tracker")

	var podsLister *kubelet.PodsLister
	var err error

	switch c.DeploySystem {
	case "porto":
		// TODO: add porto support.
		return nil, fmt.Errorf("unfortunately we don't support porto yet")
	case "kubernetes", "k8s":
		var kubeletOverrides kubelet.KubeletSettingsOverrides

		kubeletOverrides.CgroupDriver = c.KubernetesConfig.KubeletCgroupDriver
		kubeletOverrides.CgroupRoot = c.KubernetesConfig.KubeletCgroupRoot
		kubeletOverrides.CgroupsQOSMode = c.KubernetesConfig.KubeletCgroupQOSMode
		kubeletOverrides.CgroupContainerPrefix = c.KubernetesConfig.KubeletCgroupContainerPrefix
		kubeletOverrides.KubernetesAPIServerHost = c.KubernetesConfig.KubernetesAPIServerHost

		podsLister, err = kubelet.NewPodsLister(xlog.Wrap(l), c.KubernetesConfig.TopologyLableKey, kubeletOverrides, cgroupPrefix)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unspecified deploy system, must be one of [porto, kubernetes (k8s)], got: %s", c.DeploySystem)
	}

	blackListRegexps := []*blackListEntryCompiled{}
	whiteListRegexps := []*regexp.Regexp{}

	for _, blackListItem := range c.PodOptions.DenyList {
		reg, err := regexp.Compile(blackListItem.Pattern)
		if err != nil {
			l.Warn(
				"Failed to compile blacklist regexp",
				log.String("pattern", blackListItem.Pattern),
			)
			continue
		}

		blackListRegexps = append(
			blackListRegexps,
			&blackListEntryCompiled{
				Regexp: reg,
				Reason: blackListItem.Reason,
			},
		)
	}

	for _, whiteListPattern := range c.PodOptions.AllowList {
		reg, err := regexp.Compile(whiteListPattern.Pattern)
		if err != nil {
			l.Warn(
				"Failed to compile whitelist regexp",
				log.String("pattern", whiteListPattern.Pattern),
			)
			continue
		}

		whiteListRegexps = append(whiteListRegexps, reg)
	}

	cpuModel, err := cpuinfo.GetCPUModel()
	if err != nil {
		return nil, err
	}

	podsCgroupTracker := &PodsCgroupTracker{
		l:                l,
		conf:             c,
		podsLister:       podsLister,
		hostname:         podsLister.GetHost(),
		cpuModelName:     cpuModel,
		blackListRegexps: blackListRegexps,
		whiteListRegexps: whiteListRegexps,
	}

	return podsCgroupTracker, nil
}

//////////////////////////////////////////////////////////////

func (p *Profiler) updatePodsCgroups(ctx context.Context) error {
	cgroups, err := p.podsCgroupTracker.refreshCgroups(ctx)
	if err != nil {
		return fmt.Errorf("refreshing cgroups: %w", err)
	}

	if err := p.TraceCgroups(cgroups); err != nil {
		return fmt.Errorf("tracing cgroups: %w", err)
	}
	return nil
}

func (p *Profiler) runPodsCgroupTracker(ctx context.Context) error {
	err := p.podsCgroupTracker.podsLister.Init(ctx)
	if err != nil {
		return fmt.Errorf("failed to initialize pods lister: %w", err)
	}

	err = p.updatePodsCgroups(ctx)
	if err != nil {
		return err
	}

	tick := time.NewTicker(p.conf.PodsDeploySystemConfig.UpdateCgroupsPeriod)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
			p.log.Info("Run update traced cgroups")
			err = p.updatePodsCgroups(ctx)
			if err != nil {
				return err
			}
		}
	}
}
