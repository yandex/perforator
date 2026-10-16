//go:build cgo

package cprofile

import (
	"bytes"
	"runtime"
	"testing"

	pprof "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	pb "github.com/yandex/perforator/perforator/proto/profile"
)

func TestMergeSessionKeepsManagerAlive(t *testing.T) {
	session := func() *MergeSession {
		manager, err := NewMergeManager(1)
		require.NoError(t, err)
		session, err := manager.Start(&pb.MergeOptions{})
		require.NoError(t, err)
		return session
	}()
	defer session.Close()
	require.NotNil(t, session.manager)

	input := &pprof.Profile{
		SampleType: []*pprof.ValueType{{Type: "cpu", Unit: "cycles"}},
		Sample:     []*pprof.Sample{{Value: []int64{7}}},
	}
	var data bytes.Buffer
	require.NoError(t, input.WriteUncompressed(&data))
	for i := 0; i < 3; i++ {
		runtime.GC()
		require.NoError(t, session.AddPProfProfile(data.Bytes()))
	}
	runtime.GC()
	merged, err := session.Finish()
	require.NoError(t, err)
	defer merged.Free()
	serialized, err := merged.MarshalPProf()
	require.NoError(t, err)
	output, err := pprof.ParseData(serialized)
	require.NoError(t, err)
	var total int64
	for _, sample := range output.Sample {
		total += sample.Value[0]
	}
	require.Equal(t, int64(21), total)
	session.Close()
	require.Nil(t, session.manager)
}
