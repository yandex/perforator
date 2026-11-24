package kubelet

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	kube "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func assertEncode(t *testing.T, w io.Writer, data any) {
	t.Helper()
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		t.Errorf("unexpected error while writing JSON: %v", err)
	}
}

func TestGetPods(t *testing.T) {
	fakePods := kube.PodList{
		Items: []kube.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{UID: "1231314", Name: "pod1"},
				Status:     kube.PodStatus{Phase: kube.PodRunning, QOSClass: kube.PodQOSGuaranteed},
			},
		},
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		assertEncode(t, w, fakePods)
	}))
	defer server.Close()

	client := &http.Client{}

	podLister := &PodsLister{
		logger:       xlog.ForTest(t),
		nodeName:     "test-node",
		nodeURL:      server.URL,
		client:       client,
		cgroupPrefix: "/tmp/fake-cgroups",
	}

	pods, err := podLister.getPods(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(pods) != 1 {
		t.Fatalf("expected 1 pod, got %d", len(pods))
	}
}

func TestResolveKubeletContainerPrefix(t *testing.T) {
	// Create a temporary directory to act as our fake cgroup prefix.
	tmpDir, err := os.MkdirTemp("", "cgroups")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	fakePods := kube.PodList{
		Items: []kube.Pod{
			{
				ObjectMeta: metav1.ObjectMeta{UID: "2415121", Name: "pod1"},
				Status:     kube.PodStatus{Phase: kube.PodRunning, QOSClass: kube.PodQOSBestEffort},
			},
		},
	}

	fakePodCgroup := filepath.Join(tmpDir, "kubepods", "besteffort", fmt.Sprintf("pod%v", fakePods.Items[0].ObjectMeta.UID))
	err = os.MkdirAll(fakePodCgroup, 0755)
	if err != nil {
		t.Fatal(err)
	}

	fakeContainerDir := filepath.Join(fakePodCgroup, "docker-4b11478133fedf541bc8234b41a03b026161d31415e36c6e8775a90bca10f31d")
	err = os.Mkdir(fakeContainerDir, 0755)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertEncode(t, w, fakePods)
	}))
	defer server.Close()

	client := &http.Client{}

	podLister := &PodsLister{
		logger:       xlog.ForTest(t),
		nodeName:     "test-node",
		nodeURL:      server.URL,
		client:       client,
		cgroupPrefix: filepath.Join(tmpDir, "kubepods"),
	}

	err = podLister.resolveKubeletContainerPrefix(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if podLister.kubeletSettings.containerPrefix == "" {
		t.Fatalf("expected a resolved container prefix, got empty string")
	}
}

func TestTryResolveContainerPrefixFromContainerRuntime(t *testing.T) {
	cases := []struct {
		name     string
		endpoint string
		expected string
	}{
		{"containerd", "unix:///var/run/containerd/containerd.sock", containerdPrefix},
		{"crio", "unix:///var/run/crio/crio.sock", crioPrefix},
		{"cri-dockerd", "unix:///var/run/cri-dockerd.sock", criDockerdPrefix},
		{"unknown", "unix:///var/run/unknown.sock", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := kubeletConfig{
				ContainerRuntimeEndpoint: tc.endpoint,
			}
			got := tryResolveContainerPrefixFromContainerRuntime(cfg)
			if got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
