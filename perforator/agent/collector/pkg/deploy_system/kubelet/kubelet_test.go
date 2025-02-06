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
		cgroupRoot      string
		uid             types.UID
		qosClass        v1.PodQOSClass
		systemDRewrites bool
		expected        string
	}{
		{
			name:            "test",
			cgroupRoot:      "/kubepods",
			uid:             "foo",
			qosClass:        v1.PodQOSBestEffort,
			systemDRewrites: false,
			expected:        "/kubepods/besteffort/podfoo",
		},
		{
			name:            "test",
			cgroupRoot:      "/kubepods.slice",
			uid:             "foo",
			qosClass:        v1.PodQOSBestEffort,
			systemDRewrites: true,
			expected:        "/kubepods.slice/kubepods-besteffort.slice/kubepods-besteffort-podfoo.scope",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			settings := kubeletCgroupSettings{
				root:    tc.cgroupRoot,
				systemd: tc.systemDRewrites,
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
