package kubelet

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"

	v1 "k8s.io/api/core/v1"

	deploysystemmodel "github.com/yandex/perforator/perforator/agent/collector/pkg/deploy_system/model"
	"github.com/yandex/perforator/perforator/internal/kubeletclient"
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
	labels      map[string]string
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
	client                   *kubeletclient.Client
	nodeName                 string
	nodeURL                  string
	kubeletSettingsOverrides KubeletSettingsOverrides
	kubeletSettings          kubeletCgroupSettings

	// In most cases equals to the value of topology.kubernetes.io/zone lable. See https://kubernetes.io/docs/reference/labels-annotations-taints/#topologykubernetesiozone
	topology string
}

func (p *PodsLister) GetHost() string {
	return p.nodeName
}

// Implements deploysystemmodel.PodsLister
func (p *PodsLister) List() ([]deploysystemmodel.Pod, error) {
	pods, err := p.getPods()
	if err != nil {
		return nil, err
	}

	res := make([]deploysystemmodel.Pod, 0, len(pods))
	for _, pod := range pods {
		// Only running pods have cgoups.
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
			containerCgroup := parts[1]
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

		service, err := getOwner(&pod)
		if err != nil {
			return nil, err
		}

		res = append(res, &Pod{
			name:        pod.Name,
			topology:    p.topology,
			containers:  containers,
			cgroupName:  cgroup,
			labels:      pod.Labels,
			serviceName: service,
		})
	}

	return res, nil
}

func NewPodsLister(topologyLableKey string, kubeletSettingsOverrides KubeletSettingsOverrides) (*PodsLister, error) {
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

	// Otherwise we get an error: SSL certificate problem: self-signed certificate in certificate chain.
	// Failed to verify the legitimacy of the server and therefore could not establish a secure connection to it.
	// By default the kubelet serving certificate deployed by kubeadm is self-signed:
	// https://kubernetes.io/docs/tasks/administer-cluster/kubeadm/kubeadm-certs/#:~:text=By%20default%20the%20kubelet%20serving%20certificate%20deployed%20by%20kubeadm%20is%20self%2Dsigned
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	httpclient := &http.Client{Transport: tr}
	client := kubeletclient.New(httpclient)

	podLister := &PodsLister{
		nodeName:                 name,
		nodeURL:                  url + "/pods",
		client:                   client,
		kubeletSettingsOverrides: kubeletSettingsOverrides,
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
	if p.kubeletSettingsOverrides.CgroupRoot == "" {
		resolveNeeded = true
	}
	var resolved kubeletCgroupSettings
	if resolveNeeded {
		var err error
		resolved, err = resolveKubeletCgroupSettings(ctx, p.client)
		if err != nil {
			return fmt.Errorf("failed to detect kubelet cgroup root: %w", err)
		}
	}
	if p.kubeletSettingsOverrides.CgroupDriver != "" {
		if p.kubeletSettingsOverrides.CgroupDriver == "systemd" {
			resolved.systemd = true
		} else if p.kubeletSettingsOverrides.CgroupDriver != "cgroupfs" {
			return fmt.Errorf("invalid value for cgroup driver override (expected cgroupfs or systemd): %q", p.kubeletSettingsOverrides.CgroupDriver)
		}
	}
	if p.kubeletSettingsOverrides.CgroupRoot != "" {
		resolved.root = p.kubeletSettingsOverrides.CgroupRoot
	}
	p.kubeletSettings = resolved
	return nil
}
