package kubelet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	kube "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/slices"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

const (
	kubeletPort = "10250"
	nodeEnv     = "NODE_NAME"
	nodeIP      = "NODE_IP"

	// See https://kubernetes.io/docs/tasks/run-application/access-api-from-pod/#directly-accessing-the-rest-api
	kubernetesAPIServerHostFallback = "kubernetes.default.svc.cluster.local"
	kubernetesAPIServerHostEnv      = "KUBERNETES_SERVICE_HOST"
	kubernetesAPIServerPortEnv      = "KUBERNETES_SERVICE_PORT"

	getPodsRequestTimeout = 10 * time.Second

	containerdPrefix = "cri-containerd-" // https://github.com/containerd/containerd/blob/59c8cf6ea5f4175ad512914dd5ce554942bf144f/internal/cri/server/podsandbox/helpers_linux.go#L67
	crioPrefix       = "crio-"           // https://github.com/cri-o/cri-o/blob/086f182d7d883326159b165d9b5958ae2ff53e14/internal/config/cgmgr/cgmgr_linux.go#L23
	criDockerdPrefix = "docker-"         // https://github.com/Mirantis/cri-dockerd/blob/372c8f747e45b976b4b69a4023a69438e84f7e23/core/sandbox_helpers.go#L54
)

var qosClassToCgroupSubstr = map[kube.PodQOSClass]string{
	kube.PodQOSGuaranteed: "guaranteed",
	kube.PodQOSBestEffort: "besteffort",
	kube.PodQOSBurstable:  "burstable",
}

func getKubernetesAPIServerHost() string {
	host := os.Getenv(kubernetesAPIServerHostEnv)
	port := os.Getenv(kubernetesAPIServerPortEnv)

	if host != "" && port != "" {
		return fmt.Sprintf("%s:%s", host, port)
	}

	return kubernetesAPIServerHostFallback
}

func getNodeName() (string, error) {
	node := os.Getenv(nodeEnv)
	if node == "" {
		return "", fmt.Errorf("could not get node name: expected environment variable %s", nodeEnv)
	}

	return node, nil
}

func getNodeURL() (string, error) {
	var host string
	ip := os.Getenv(nodeIP)
	if ip != "" {
		if strings.Contains(ip, ":") {
			// ipv6 address
			ip = fmt.Sprintf("[%s]", ip)
		}
		host = ip
	} else {
		name, err := getNodeName()
		if err != nil {
			return "", fmt.Errorf("can't get node url %w", err)
		}
		host = name
	}
	url := fmt.Sprintf("https://%s:%s", host, kubeletPort)

	return url, nil
}

func (p *PodsLister) getPods() ([]kube.Pod, error) {
	ctx, cancel := context.WithTimeout(context.Background(), getPodsRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.nodeURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("couldn't fetch pods on %s, got error: %w", p.nodeURL, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body during fetching pods: %w", err)
	}

	var podList kube.PodList
	err = json.Unmarshal(body, &podList)
	if err != nil {
		return nil, fmt.Errorf("error unmarshalling pods responce: %w", err)
	}

	return podList.Items, nil
}

type kubeletConfigWrapper struct {
	Config kubeletConfig `json:"kubeletconfig"`
}

type kubeletConfig struct {
	CgroupRoot               string `json:"cgroupRoot"`
	CgroupDriver             string `json:"cgroupDriver"`
	CgroupsPerQOS            bool   `json:"cgroupsPerQOS"`
	ContainerRuntimeEndpoint string `json:"containerRuntimeEndpoint"`
}

type KubeletSettingsOverrides struct {
	CgroupRoot            []string `json:"cgroupRoot"`
	CgroupDriver          string   `json:"cgroupDriver"`
	CgroupsQOSMode        string   `json:"cgroupsQOSMode"`
	CgroupContainerPrefix string   `json:"cgroupContainerPrefix"`

	KubernetesAPIServerHost string `json:"kubernetesAPIServerHost"`
}

type cgroupsQOSMode int

const (
	// best-effort and burstable classes have subdirectories
	cgroupsQOSModeNotGuaranteed cgroupsQOSMode = iota
	// qos has no effect on cgroup name
	cgroupsQOSModeNone
	// all modes have subdirectories
	// TODO: check if this mode is possible at all
	cgroupsQOSModeAll
)

type kubeletCgroupSettings struct {
	root            []string
	systemd         bool
	qosMode         cgroupsQOSMode
	containerPrefix string
}

func (p *PodsLister) resolveKubeletContainerPrefix() error {
	// Try to derive from the container's cgroup
	pods, err := p.getPods()
	if err != nil {
		return err
	}
	if len(pods) == 0 {
		return fmt.Errorf("couldn't get kubernetes pods, the pod list is empty")
	}

	var (
		containerPrefix string
		cgroupFullPath  string
		lasError        error
	)
podLoop:
	for _, pod := range pods {
		cgroup, err := buildCgroup(&p.kubeletSettings, podInfo{
			UID:      pod.ObjectMeta.UID,
			QOSClass: pod.Status.QOSClass,
		})
		if err != nil {
			lasError = err
			continue
		}
		cgroupFullPath = filepath.Join(p.cgroupPrefix, cgroup)

		entries, err := os.ReadDir(cgroupFullPath)
		if err != nil {
			lasError = err
			continue
		}

		pattern := regexp.MustCompile(`([a-z0-9]{64})`)
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if pattern.MatchString(name) {
				idx := strings.LastIndex(name, "-")
				containerPrefix = name[:idx+1]
				break podLoop
			}
		}
	}

	if containerPrefix != "" {
		p.kubeletSettings.containerPrefix = containerPrefix
		return nil
	}

	return fmt.Errorf("container prefix is empty, last checked pod cgroup: %s; last error: %w", cgroupFullPath, lasError)
}

func tryResolveContainerPrefixFromContainerRuntime(config kubeletConfig) string {
	// Try to resolve via well-known container runtime endpoints
	// https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/install-kubeadm/#installing-runtime
	switch config.ContainerRuntimeEndpoint {
	case "unix:///var/run/containerd/containerd.sock":
		return containerdPrefix
	case "unix:///var/run/crio/crio.sock":
		return crioPrefix
	case "unix:///var/run/cri-dockerd.sock":
		return criDockerdPrefix
	}

	return ""
}

func resolveKubeletCgroupSettings(ctx context.Context, client *http.Client) (kubeletCgroupSettings, error) {
	var s kubeletCgroupSettings
	url, err := getNodeURL()
	if err != nil {
		return s, fmt.Errorf("failed to resolve kubelet API endpoint: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/configz", url), nil)
	if err != nil {
		return s, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return s, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return s, fmt.Errorf("error reading /configz response body: %w", err)
	}

	var config kubeletConfigWrapper
	err = json.Unmarshal(body, &config)
	if err != nil {
		return s, fmt.Errorf("error unmarshalling /configz response body: %w", err)
	}
	s.root = strings.Split(config.Config.CgroupRoot, "/")
	s.root = slices.Filter(s.root, func(s string) bool { return s != "" })

	if config.Config.CgroupsPerQOS {
		s.qosMode = cgroupsQOSModeNotGuaranteed
		s.root = append(s.root, "kubepods") // https://github.com/kubernetes/kubernetes/blob/aa08c90fca8d30038d3f05c0e8f127b540b40289/pkg/kubelet/cm/container_manager_linux.go#L255
	} else {
		s.qosMode = cgroupsQOSModeNone
	}

	if config.Config.CgroupDriver == "systemd" {
		s.systemd = true
		s.containerPrefix = tryResolveContainerPrefixFromContainerRuntime(config.Config)
	} else if config.Config.CgroupDriver != "cgroupfs" {
		return kubeletCgroupSettings{}, fmt.Errorf("unsupported cgroup driver %q (expected systemd or cgroupfs)", config.Config.CgroupDriver)
	}

	return s, nil
}

func (p *PodsLister) getTopology(topologyLableKey string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), getPodsRequestTimeout)
	defer cancel()

	url := fmt.Sprintf("https://%s/api/v1/nodes/%s", p.apiServerHost, p.nodeName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("couldn't fetch node info on %s, got error: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body during fetching node info: %w", err)
	}

	var node kube.Node
	err = json.Unmarshal(body, &node)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling node info responce: %w", err)
	}

	return node.Labels[topologyLableKey], nil
}

func getOwner(ctx context.Context, logger xlog.Logger, pod *kube.Pod) (string, error) {
	if len(pod.OwnerReferences) == 0 || !*pod.OwnerReferences[0].Controller {
		return pod.ObjectMeta.Name, nil
	}

	switch pod.OwnerReferences[0].Kind {
	case "ReplicaSet":
		// Most likely it is replicaSet owned by Deployment, so we trim the end hash of replica set. Example: kube-dns-autoscaler-6c555f9587
		// There might be a better way with api-server client, see https://stackoverflow.com/questions/67473802/how-can-i-find-a-pods-controller-deployment-daemonset-using-the-kubernetes-go
		name := pod.OwnerReferences[0].Name
		idx := strings.LastIndex(name, "-")
		if idx == -1 {
			return name, nil
		}

		_, err := strconv.ParseUint(name[idx+1:], 16, 64)
		if err != nil {
			return name, nil
		}

		return name[:idx], nil
	case "DaemonSet", "StatefulSet", "Job":
	default:
		// TODO: this warning will fire for custom controllers, it looks unfortunate
		logger.Warn(
			ctx,
			"unknown resource manager for the pod",
			log.String("ns", pod.ObjectMeta.Namespace),
			log.String("pod", pod.ObjectMeta.Name),
			log.String("kind", pod.OwnerReferences[0].Kind),
		)
	}
	return pod.OwnerReferences[0].Name, nil

}

// podInfo is subset of v1.Pod enough to derive cgroup names
type podInfo struct {
	// UID is .ObjectMeta.UID
	UID types.UID
	// QOSClass is .Status.QOSClass
	QOSClass kube.PodQOSClass
}

func makeSystemDPath(parts []string) string {
	// see https://github.com/kubernetes/kubernetes/blob/491a23f0793a16a3036d17494c29b7a403b604d6/pkg/kubelet/cm/cgroup_manager_linux.go#L69
	// and https://github.com/kubernetes/kubernetes/blob/491a23f0793a16a3036d17494c29b7a403b604d6/pkg/kubelet/cm/cgroup_manager_linux.go#L82
	// TODO: probably we should just call that function instead
	var acc string
	var converted []string
	for _, part := range parts {
		part = strings.Replace(part, "-", "_", -1)
		if acc != "" {
			part = acc + "-" + part
		}
		converted = append(converted, part+".slice")
		acc = part
	}
	return path.Join(converted...)
}

func buildCgroup(settings *kubeletCgroupSettings, pod podInfo) (string, error) {
	podUID := string(pod.UID)
	var includeQOS bool
	switch settings.qosMode {
	case cgroupsQOSModeAll:
		includeQOS = true
	case cgroupsQOSModeNotGuaranteed:
		includeQOS = pod.QOSClass != kube.PodQOSGuaranteed
	case cgroupsQOSModeNone:
		includeQOS = false
	default:
		return "", fmt.Errorf("error building pod's cgroup: unknown qosMode: %v", settings.qosMode)
	}

	podName := "pod" + podUID

	var nameParts []string

	if includeQOS {
		podQOSClass, ok := qosClassToCgroupSubstr[pod.QOSClass]
		if !ok {
			return "", fmt.Errorf("error building pod's cgroup: got unknown PodQOSClass: %v. Pod's UID: %v", pod.QOSClass, pod.UID)
		}
		nameParts = append(settings.root[:], podQOSClass, podName)
	} else {
		nameParts = append(settings.root[:], podName)
	}

	if settings.systemd {
		return "/" + makeSystemDPath(nameParts), nil
	} else {
		return "/" + path.Join(nameParts...), nil
	}
}
