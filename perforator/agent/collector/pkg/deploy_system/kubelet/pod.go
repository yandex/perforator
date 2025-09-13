package kubelet

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	v1 "k8s.io/api/core/v1"

	deploysystemmodel "github.com/yandex/perforator/perforator/agent/collector/pkg/deploy_system/model"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	// See https://kubernetes.io/docs/reference/labels-annotations-taints/#topologykubernetesiozone
	defaultTopologyLableKey = "topology.kubernetes.io/zone"
)

type kubeContainer struct {
	name   string
	cgroup string
}

func (c *kubeContainer) Name() string {
	return c.name
}

func (c *kubeContainer) CgroupBaseName() string {
	return c.cgroup
}

type Pod struct {
	name        string                        // kubernetes name of the pod
	topology    string                        // pod topology
	containers  []deploysystemmodel.Container // containers that are in this pod
	cgroupName  string                        // cgroup name like kubepods/burstable/podf8448eeb-fdf5-4fb4-9791-33ed68005ee9
	labels      map[string]string             // contains namespace label and labels defined in pod's Kubernetes manifests
	serviceName string
}

// Implements deploysystemmodel.Pod
func (p *Pod) ID() string {
	return p.name
}

// Implements deploysystemmodel.Pod
func (p *Pod) Topology() string {
	return p.topology
}

// Implements deploysystemmodel.Pod
func (p *Pod) Labels() map[string]string {
	if p.labels != nil {
		return p.labels
	}

	return map[string]string{}
}

// Implements deploysystemmodel.Pod
func (p *Pod) CgroupName() string {
	return p.cgroupName
}

// Implements deploysystemmodel.Pod
func (p *Pod) Containers() []deploysystemmodel.Container {
	return p.containers
}

// Implements deploysystemmodel.Pod
func (p *Pod) ServiceName() string {
	return p.serviceName
}

// Implements deploysystemmodel.Pod
func (p *Pod) IsPerforatorEnabled() (*bool, string) {
	// TODO:
	return nil, ""
}

type PodsLister struct {
	logger                   xlog.Logger
	client                   *http.Client
	nodeName                 string
	nodeURL                  string
	kubeletSettingsOverrides KubeletSettingsOverrides
	kubeletSettings          kubeletCgroupSettings

	// In most cases equals to the value of topology.kubernetes.io/zone lable. See https://kubernetes.io/docs/reference/labels-annotations-taints/#topologykubernetesiozone
	topology string

	// Kubernetes api server host. See https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#directly-accessing-the-rest-api
	apiServerHost string

	// E.g., "/sys/fs/cgroup"
	cgroupPrefix string
}

func (p *PodsLister) GetHost() string {
	return p.nodeName
}

// Implements deploysystemmodel.PodsLister
func (p *PodsLister) List(ctx context.Context) ([]deploysystemmodel.Pod, error) {
	pods, err := p.getPods()
	if err != nil {
		return nil, err
	}

	res := make([]deploysystemmodel.Pod, 0, len(pods))
	for _, pod := range pods {
		// Only running pods have cgroups.
		if pod.Status.Phase != v1.PodRunning {
			continue
		}

		containers := make([]deploysystemmodel.Container, 0, len(pod.Spec.Containers))
		for _, container := range pod.Status.ContainerStatuses {
			// containerd://4b11478133fedf541bc8234b41a03b026161d31415e36c6e8775a90bca10f31d
			parts := strings.SplitN(container.ContainerID, "//", 2)
			if len(parts) != 2 {
				continue
			}
			containerCgroup := p.kubeletSettings.containerPrefix + parts[1]
			if p.kubeletSettings.systemd {
				containerCgroup = containerCgroup + ".scope"
			}

			containers = append(containers, &kubeContainer{
				name:   container.Name,
				cgroup: containerCgroup,
			})
		}

		cgroup, err := buildCgroup(&p.kubeletSettings, podInfo{
			UID:      pod.ObjectMeta.UID,
			QOSClass: pod.Status.QOSClass,
		})
		if err != nil {
			return nil, err
		}

		service, err := getOwner(ctx, p.logger, &pod)
		if err != nil {
			return nil, err
		}

		labels := make(map[string]string)
		for k, v := range pod.ObjectMeta.Labels {
			labels[SanitizeLabelName(k)] = v
		}
		labels["namespace"] = pod.ObjectMeta.Namespace

		res = append(res, &Pod{
			name:        pod.Name,
			topology:    p.topology,
			containers:  containers,
			cgroupName:  cgroup,
			labels:      labels,
			serviceName: service,
		})
	}

	return res, nil
}

func NewPodsLister(logger xlog.Logger, topologyLableKey string, kubeletSettingsOverrides KubeletSettingsOverrides, cgroupPrefix string) (*PodsLister, error) {
	if topologyLableKey == "" {
		topologyLableKey = defaultTopologyLableKey
	}
	name, err := getNodeName()
	if err != nil {
		return nil, err
	}

	url, err := getNodeURL()
	if err != nil {
		return nil, err
	}

	apiServerHost := kubeletSettingsOverrides.KubernetesAPIServerHost
	if apiServerHost == "" {
		apiServerHost = getKubernetesAPIServerHost()
	}

	// Otherwise we get an error: SSL certificate problem: self-signed certificate in certificate chain.
	// Failed to verify the legitimacy of the server and therefore could not establish a secure connection to it.
	// By default the kubelet serving certificate deployed by kubeadm is self-signed:
	// https://kubernetes.io/docs/tasks/administer-cluster/kubeadm/kubeadm-certs/#:~:text=By%20default%20the%20kubelet%20serving%20certificate%20deployed%20by%20kubeadm%20is%20self%2Dsigned
	tr := &kubeletTokenTransport{
		rt: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	client := &http.Client{Transport: tr}

	podLister := &PodsLister{
		logger:                   logger,
		nodeName:                 name,
		nodeURL:                  url + "/pods",
		apiServerHost:            apiServerHost,
		client:                   client,
		kubeletSettingsOverrides: kubeletSettingsOverrides,
		cgroupPrefix:             cgroupPrefix,
	}

	topology, err := podLister.getTopology(topologyLableKey)
	if err != nil {
		return nil, err
	}
	podLister.topology = topology

	return podLister, nil
}

func (p *PodsLister) Init(ctx context.Context) error {
	var resolveNeeded bool
	if p.kubeletSettingsOverrides.CgroupDriver == "" {
		resolveNeeded = true
	}
	if len(p.kubeletSettingsOverrides.CgroupRoot) == 0 {
		resolveNeeded = true
	}
	if p.kubeletSettingsOverrides.CgroupsQOSMode == "" {
		resolveNeeded = true
	}
	if p.kubeletSettingsOverrides.CgroupDriver == "systemd" && p.kubeletSettingsOverrides.CgroupContainerPrefix == "" {
		resolveNeeded = true
	}
	var resolved kubeletCgroupSettings
	if resolveNeeded {
		var err error
		resolved, err = resolveKubeletCgroupSettings(ctx, p.client)
		if err != nil {
			return fmt.Errorf("failed to resolve kubelet cgroup settings: %w", err)
		}
	}
	if p.kubeletSettingsOverrides.CgroupDriver != "" {
		if p.kubeletSettingsOverrides.CgroupDriver == "systemd" {
			resolved.systemd = true
		} else if p.kubeletSettingsOverrides.CgroupDriver != "cgroupfs" {
			return fmt.Errorf("invalid value for cgroup driver override (expected cgroupfs or systemd): %q", p.kubeletSettingsOverrides.CgroupDriver)
		}
	}
	if len(p.kubeletSettingsOverrides.CgroupRoot) > 0 {
		resolved.root = p.kubeletSettingsOverrides.CgroupRoot
	}
	switch p.kubeletSettingsOverrides.CgroupsQOSMode {
	case "none":
		resolved.qosMode = cgroupsQOSModeNone
	case "not-guaranteed":
		resolved.qosMode = cgroupsQOSModeNotGuaranteed
	case "all":
		resolved.qosMode = cgroupsQOSModeAll
	case "":
	default:
		return fmt.Errorf("invalid value for cgroups qos mode override (expected none, not-guaranteed or all): %q", p.kubeletSettingsOverrides.CgroupsQOSMode)
	}
	if p.kubeletSettingsOverrides.CgroupContainerPrefix != "" {
		resolved.containerPrefix = p.kubeletSettingsOverrides.CgroupContainerPrefix
	}

	p.kubeletSettings = resolved
	if p.kubeletSettings.containerPrefix == "" && p.kubeletSettings.systemd {
		err := p.resolveKubeletContainerPrefix()
		if err != nil {
			return fmt.Errorf("couldn't resolve container prefix %w", err)
		}
	}

	return nil
}
