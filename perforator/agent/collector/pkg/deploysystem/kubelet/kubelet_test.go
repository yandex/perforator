package kubelet

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

//go:embed kubelet-configz-response.json
var kubelerConfigzResponse string

func TestParse(t *testing.T) {
	var conf kubeletConfigWrapper
	err := json.Unmarshal([]byte(kubelerConfigzResponse), &conf)
	assert.NoError(t, err)
	assert.Equal(t, "cgroupfs", conf.Config.CgroupDriver)
	assert.Equal(t, "/", conf.Config.CgroupRoot)
}

func TestBuildCgroup(t *testing.T) {
	tests := []struct {
		name            string
		cgroupRoot      []string
		qosMode         cgroupsQOSMode
		uid             types.UID
		qosClass        v1.PodQOSClass
		systemDRewrites bool
		expected        string
	}{
		{
			name:            "simple-cgroupfs",
			cgroupRoot:      []string{"kubepods"},
			uid:             "foo",
			qosClass:        v1.PodQOSBestEffort,
			systemDRewrites: false,
			expected:        "/kubepods/besteffort/podfoo",
		},

		{
			name:       "cgroupsPerQOSDisabled",
			cgroupRoot: []string{"kubepods"},
			uid:        "bar",
			qosMode:    cgroupsQOSModeNone,
			qosClass:   v1.PodQOSBurstable,
			expected:   "/kubepods/podbar",
		},
		{
			name:            "simple-systemd",
			cgroupRoot:      []string{"kubepods"},
			uid:             "foo",
			qosClass:        v1.PodQOSBestEffort,
			systemDRewrites: true,
			expected:        "/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-podfoo.slice",
		},
		{
			name:            "kind",
			cgroupRoot:      []string{"kubelet", "kubepods"},
			uid:             "c9b109d4_2626_4f72_9301_3bfe505113e6",
			qosClass:        v1.PodQOSBestEffort,
			systemDRewrites: true,
			expected:        "/kubelet.slice/kubelet-kubepods.slice/kubelet-kubepods-besteffort.slice/kubelet-kubepods-besteffort-podc9b109d4_2626_4f72_9301_3bfe505113e6.slice",
		},
		{
			name:       "guaranteed",
			cgroupRoot: []string{"kubepods"},
			uid:        "bar",
			qosClass:   v1.PodQOSGuaranteed,
			expected:   "/kubepods/podbar",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			settings := kubeletCgroupSettings{
				root:    tc.cgroupRoot,
				systemd: tc.systemDRewrites,
				qosMode: tc.qosMode,
			}
			s, err := buildCgroup(&settings, podInfo{
				UID:      tc.uid,
				QOSClass: tc.qosClass,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.expected, s)
		})
	}
}
