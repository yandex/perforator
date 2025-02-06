package profiler

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/yandex/perforator/library/go/core/buildinfo"
	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/deploy_system/kubelet"
	deploysystemmodel "github.com/yandex/perforator/perforator/agent/collector/pkg/deploy_system/model"
	"github.com/yandex/perforator/perforator/pkg/linux/cpuinfo"
)

const (
	updateCgroupPeriod = 1 * time.Minute
)

type blackListEntryCompiled struct {
	Regexp *regexp.Regexp
	Reason string
}

type containerInfo struct {
	podServiceName string
	containerName  string
}

type PodsCgroupTracker struct {
	l    log.Logger
	conf *config.PodsDeploySystemConfig

	podsLister   deploysystemmodel.PodsLister
	cpuModelName string
	hostname     string

	workloadsmu sync.RWMutex
	// key is container cgroup name
	workloads map[string]*containerInfo

	blackListRegexps []*blackListEntryCompiled
	whiteListRegexps []*regexp.Regexp
}

func (t *PodsCgroupTracker) buildCgroupConfig(
	pod deploysystemmodel.Pod,
) *CgroupConfig {
	labels := t.makeCommonProfileLabels()
	labels["pod"] = pod.ID()
	labels["cluster"] = pod.Topology()
	labels["service"] = pod.ServiceName()
	labels["cgroup"] = pod.CgroupName()

	cgroupConfig := &CgroupConfig{
		Name:   pod.CgroupName(),
		Labels: labels,
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

	for key, val := range t.conf.Labels {
		labels[key] = val
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

func (t *PodsCgroupTracker) isPodFiltered(pod deploysystemmodel.Pod) (bool, string) {
	if flag, message := pod.IsPerforatorEnabled(); flag != nil {
		return *flag, message
	}

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

func (t *PodsCgroupTracker) rebuildWorkloadInfo(pods []deploysystemmodel.Pod) {
	workloads := make(map[string]*containerInfo)
	t.l.Info("Building workload mapping", log.Int("count", len(pods)))
	for _, pod := range pods {
		for _, cont := range pod.Containers() {
			t.l.Info(
				"Adding container",
				log.String("cgroup", cont.CgroupBaseName()),
				log.String("pod", pod.ID()),
				log.String("container", cont.Name()),
			)
			workloads[cont.CgroupBaseName()] = &containerInfo{
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
	pods, err := t.podsLister.List()
	if err != nil {
		return nil, err
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

func (t *PodsCgroupTracker) ResolveWorkload(cgroups []string) ([]string, bool) {
	t.l.Info("Resolving workload", log.Strings("cgroups", cgroups))
	if len(cgroups) == 0 {
		return nil, false
	}
	t.workloadsmu.RLock()
	defer t.workloadsmu.RUnlock()
	// k8s cgroups look like this:
	// /kubepods/besteffort/pod47a861d9-65ce-4f59-9aaa-2a2268acd5e0/e6833253323a1be50457462dce9ae1315c3ce263b747fbd5a4fc96d3abfc7337
	// first part is fixed to "kubepods"
	// besteffort is scheduling QoS, we are not interested in it.
	// third part is pod id (i.e. .metadata.uid)
	// fourth part is container id (i.e. .status.containerStatus[*].containerID)
	// the first three are skipped as they are parts of the tracked cgroup
	info, ok := t.workloads[cgroups[0]]
	if !ok {
		t.l.Warn("Workload not found", log.String("key", cgroups[0]))
		return nil, false
	}
	t.l.Info("Workload found", log.String("cgroup", cgroups[0]), log.String("resolved", info.containerName))
	convertedParts := []string{
		info.podServiceName,
		info.containerName,
	}
	// usually there won't be any nested cgroups, but it is possible in the future
	// when k8s natively supports cgroup delegation
	convertedParts = append(convertedParts, cgroups[1:]...)

	return convertedParts, true
}

func newPodsCgroupTracker(c *config.PodsDeploySystemConfig, l log.Logger) (*PodsCgroupTracker, error) {
	l = l.WithName("pod_tracker")

	var podsLister deploysystemmodel.PodsLister
	var err error

	switch c.DeploySystem {
	case "porto":
		// TODO: add porto support.
		return nil, fmt.Errorf("unfortunately we don't support porto yet")
	case "kubernetes", "k8s":
		var kubeletOverrides kubelet.KubeletSettingsOverrides

		kubeletOverrides.CgroupDriver = c.KubernetesConfig.KubeletCgroupDriver
		kubeletOverrides.CgroupRoot = c.KubernetesConfig.KubeletCgroupRoot

		podsLister, err = kubelet.NewPodsLister(c.KubernetesConfig.TopologyLableKey, kubeletOverrides)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unspecified deploy system, must be one of [porto, kubernetes (k8s)], got: %s", c.DeploySystem)
	}

	blackListRegexps := []*blackListEntryCompiled{}
	whiteListRegexps := []*regexp.Regexp{}

	for _, blackListItem := range c.PodOptions.BlackList {
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

	for _, whiteListPattern := range c.PodOptions.WhiteList {
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
		return err
	}

	return p.TraceCgroups(cgroups)
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
