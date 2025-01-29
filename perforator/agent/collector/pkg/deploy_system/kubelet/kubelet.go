package kubelet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	kube "k8s.io/api/core/v1"
)

const (
	tokenPath   = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	kubeletPort = "10250"
	nodeEnv     = "NODE_NAME"

	kubernetesAPIServerHost = "kubernetes.default.svc.cluster.local"

	getPodsRequestTimeout = 10 * time.Second
)

var qosClassToCgroupSubstr = map[kube.PodQOSClass]string{
	kube.PodQOSGuaranteed: "guaranteed",
	kube.PodQOSBestEffort: "besteffort",
	kube.PodQOSBurstable:  "burstable",
}

func getNodeName() (string, error) {
	node := os.Getenv(nodeEnv)
	if node == "" {
		return "", fmt.Errorf("could not get node name: expected environment variable %s", nodeEnv)
	}

	return node, nil
}

func getNodeURL() (string, error) {
	name, err := getNodeName()
	if err != nil {
		return "", fmt.Errorf("can't get node url %w", err)
	}
	url := fmt.Sprintf("https://%s:%s/pods", name, kubeletPort)

	return url, nil
}

func (p *PodsLister) getPods() ([]kube.Pod, error) {
	// Need to read it everytime, because it might change.
	// TODO: May be add some retry policy
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't read service account token, %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), getPodsRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.nodeURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+string(token))

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

func (p *PodsLister) getTopology(topologyLableKey string) (string, error) {
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("couldn't read service account token, %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), getPodsRequestTimeout)
	defer cancel()

	url := fmt.Sprintf("https://%s/api/v1/nodes/%s", kubernetesAPIServerHost, p.nodeName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+string(token))

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

func getOwner(pod *kube.Pod) (string, error) {
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
	case "DaemonSet", "StatefulSet":
		return pod.OwnerReferences[0].Name, nil
	default:
		return "", fmt.Errorf("unknown resource manager for the pod: %v; pod name: %v", pod.OwnerReferences[0].Kind, pod.ObjectMeta.Name)
	}

}

// BuildCgroup returns unified cgroup for cgroup v2 in a format like:
// "/sys/fs/cgroup/kubepods/<POD_QOSClass>/pod<POD_UID>"
// or freezer cgroup for cgroup v1
// "/sys/fs/cgroup/freezer/kubepods/<POD_QOSClass>/pod<POD_UID>".
func buildCgroup(pod *kube.Pod) (string, error) {
	podUID := string(pod.ObjectMeta.UID)
	podQOSClass, ok := qosClassToCgroupSubstr[pod.Status.QOSClass]
	if !ok {
		return "", fmt.Errorf("error building pod's cgroup: got unknown PodQOSClass: %v. Pod's UID: %v", pod.Status.QOSClass, pod.ObjectMeta.UID)
	}

	// TODO: change agent so it can support cgroup v2, at this moment it works only with v1 with freezer hierarchy
	/* if isCgroup2UnifiedMode() {
		return path.Join("/sys/fs/cgroup/kubepods", podQOSClass, "pod"+podUID)
	}

	return path.Join("/sys/fs/cgroup/freezer/kubepods", podQOSClass, "pod"+podUID) */

	return path.Join("/kubepods", podQOSClass, "pod"+podUID), nil
}
