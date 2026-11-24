package kubelet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	kube "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/deploysystem"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func TestSourceUsesUIDAndPreservesMetadata(t *testing.T) {
	pods := kube.PodList{Items: []kube.Pod{
		{ObjectMeta: metav1.ObjectMeta{UID: "uid-a", Name: "same", Namespace: "one", Labels: map[string]string{"service": "metadata-service", "app.kubernetes.io/name": "metadata-custom"}}, Status: kube.PodStatus{Phase: kube.PodRunning, QOSClass: kube.PodQOSGuaranteed, ContainerStatuses: []kube.ContainerStatus{{Name: "app", ContainerID: "containerd://container-a"}}}},
		{ObjectMeta: metav1.ObjectMeta{UID: "uid-b", Name: "same", Namespace: "two"}, Status: kube.PodStatus{Phase: kube.PodRunning, QOSClass: kube.PodQOSGuaranteed}},
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { assertEncode(t, w, pods) }))
	defer server.Close()
	lister := &PodsLister{logger: xlog.ForTest(t), client: server.Client(), nodeName: "node", nodeURL: server.URL, kubeletSettings: kubeletCgroupSettings{root: []string{"kubepods"}}, topology: "zone"}
	source := &snapshotSource{lister: lister, initialized: true}
	snapshot, err := source.Snapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, "node", snapshot.Host)
	cache := deploysystem.NewResolver(nop.Registry{})
	require.NoError(t, cache.Publish(snapshot))
	for _, id := range []string{"uid-a", "uid-b"} {
		pod, err := cache.ResolvePod(id)
		require.NoError(t, err)
		require.Equal(t, "zone", pod.Metadata["cluster"])
		require.Contains(t, string(pod.CgroupPath), id)
		require.Equal(t, "/"+string(pod.CgroupPath), pod.Metadata["cgroup"])
		if id == "uid-a" {
			require.Equal(t, "same", pod.Metadata["pod"])
			require.Equal(t, "same", pod.Metadata["service"])
			require.Equal(t, "one", pod.Metadata["namespace"])
			require.Equal(t, "metadata-custom", pod.Metadata["pod_app_kubernetes_io_name"])
			require.Equal(t, "metadata-service", pod.Metadata["pod_service"])
		}
	}
	require.Equal(t, []deploysystem.Container{{Name: "app", CgroupBaseName: "container-a"}}, snapshot.Pods[0].Containers)
	_, err = cache.ResolvePod("same")
	require.Error(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	failed, err := source.Snapshot(ctx)
	require.Error(t, err)
	require.True(t, failed.ObservedAt.IsZero())
}

func TestSourceRejectsHTTPFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "kubelet is shutting down", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	source := &snapshotSource{initialized: true, lister: &PodsLister{client: server.Client(), nodeURL: server.URL + "/pods"}}
	snapshot, err := source.Snapshot(context.Background())
	require.ErrorContains(t, err, "GET /pods: HTTP 503: kubelet is shutting down")
	require.True(t, snapshot.ObservedAt.IsZero())
}

func TestListerDefersNetworkUntilInit(t *testing.T) {
	t.Setenv(nodeEnv, "test-node")
	t.Setenv(nodeIP, "127.0.0.1")
	lister, err := NewPodsLister(xlog.ForTest(t), "zone", KubeletSettingsOverrides{}, "/unused")
	require.NoError(t, err)
	// Replace the production transport before Init; no credentials or live API are used.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		assertEncode(t, w, map[string]string{})
	}))
	defer server.Close()
	lister.client = server.Client()
	lister.apiServerHost = strings.TrimPrefix(server.URL, "https://")
	require.ErrorContains(t, lister.Init(context.Background()), "503")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.ErrorIs(t, lister.Init(ctx), context.Canceled)
}

func TestSourceRetriesInitializationInsideSnapshot(t *testing.T) {
	topologyRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/pods":
			assertEncode(t, w, kube.PodList{})
		case "/configz":
			assertEncode(t, w, kubeletConfigWrapper{Config: kubeletConfig{CgroupDriver: "cgroupfs", CgroupRoot: "/kubepods"}})
		}
	}))
	defer server.Close()
	apiServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		topologyRequests++
		if topologyRequests == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		assertEncode(t, w, kube.Node{})
	}))
	defer apiServer.Close()
	source := &snapshotSource{lister: &PodsLister{
		logger: xlog.ForTest(t), client: apiServer.Client(), nodeURL: server.URL + "/pods",
		apiServerHost:            strings.TrimPrefix(apiServer.URL, "https://"),
		kubeletSettingsOverrides: KubeletSettingsOverrides{CgroupDriver: "cgroupfs", CgroupRoot: []string{"kubepods"}, CgroupsQOSMode: "none"},
	}}
	_, err := source.Snapshot(context.Background())
	require.ErrorContains(t, err, "503")
	for range 2 {
		snapshot, err := source.Snapshot(context.Background())
		require.NoError(t, err)
		require.Empty(t, snapshot.Pods)
	}
	require.Equal(t, 2, topologyRequests)
}
