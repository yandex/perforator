//go:build cgo

package cmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	pprof "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func writeMergeInput(t *testing.T, path string, value int64, compressed bool) {
	t.Helper()
	function := &pprof.Function{ID: 1, Name: "work", Filename: "work.cc"}
	mapping := &pprof.Mapping{ID: 1, Start: 4096, Limit: 8192, BuildID: "build", File: "program"}
	location := &pprof.Location{ID: 1, Address: 4100, Mapping: mapping, Line: []pprof.Line{{Function: function, Line: 7}}}
	p := &pprof.Profile{
		SampleType: []*pprof.ValueType{{Type: "cpu", Unit: "cycles"}, {Type: "alloc", Unit: "bytes"}},
		Sample:     []*pprof.Sample{{Location: []*pprof.Location{location}, Value: []int64{value, value * 2}}},
		Location:   []*pprof.Location{location}, Function: []*pprof.Function{function}, Mapping: []*pprof.Mapping{mapping},
		TimeNanos: value * 60e9, DurationNanos: 60e9,
	}
	var data bytes.Buffer
	if compressed {
		require.NoError(t, p.Write(&data))
	} else {
		require.NoError(t, p.WriteUncompressed(&data))
	}
	require.NoError(t, os.WriteFile(path, data.Bytes(), 0600))
}

func TestMergeProfiles500Profiles(t *testing.T) {
	dir := t.TempDir()
	paths := make([]string, 500)
	originals := make([][]byte, len(paths))
	for i := range paths {
		paths[i] = filepath.Join(dir, fmt.Sprintf("minute %d.pprof", i))
		writeMergeInput(t, paths[i], int64(i+1), i%2 == 0)
		originals[i], _ = os.ReadFile(paths[i])
	}
	var output bytes.Buffer
	require.NoError(t, mergeLocalProfiles(context.Background(), paths, 2, &output))
	merged, err := pprof.ParseData(output.Bytes())
	require.NoError(t, err)
	require.NoError(t, merged.CheckValid())
	require.Equal(t, []*pprof.ValueType{{Type: "cpu", Unit: "cycles"}, {Type: "alloc", Unit: "bytes"}}, merged.SampleType)
	var totals [2]int64
	for _, sample := range merged.Sample {
		require.Len(t, sample.Value, 2)
		totals[0] += sample.Value[0]
		totals[1] += sample.Value[1]
		require.Equal(t, "work", sample.Location[0].Line[0].Function.Name)
	}
	require.Equal(t, [2]int64{500 * 501 / 2, 500 * 501}, totals)
	for i, path := range paths {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, originals[i], data)
	}
}

func TestMergeProfilesCommand(t *testing.T) {
	path := filepath.Join(t.TempDir(), "one.pprof")
	writeMergeInput(t, path, 7, false)
	var output bytes.Buffer
	command := makeMergeCmd()
	command.SetOut(&output)
	command.SetArgs([]string{path, path})
	require.NoError(t, command.Execute())
	merged, err := pprof.ParseData(output.Bytes())
	require.NoError(t, err)
	require.NoError(t, merged.CheckValid())
	var totals [2]int64
	for _, sample := range merged.Sample {
		require.Len(t, sample.Value, 2)
		totals[0] += sample.Value[0]
		totals[1] += sample.Value[1]
	}
	require.Equal(t, [2]int64{14, 28}, totals)
}

func TestMergeProfilesErrorsDoNotWriteOutput(t *testing.T) {
	dir := t.TempDir()
	good, bad := filepath.Join(dir, "good"), filepath.Join(dir, "bad")
	writeMergeInput(t, good, 1, false)
	require.NoError(t, os.WriteFile(bad, []byte("bad profile"), 0600))
	for _, tc := range []struct {
		name  string
		paths []string
	}{
		{"corrupt", []string{good, bad}},
		{"missing file", []string{good, filepath.Join(dir, "missing")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			require.Error(t, mergeLocalProfiles(context.Background(), tc.paths, 1, &output))
			require.Empty(t, output.String())
		})
	}
}

func TestMergeProfilesArguments(t *testing.T) {
	for _, args := range [][]string{
		{}, {"unused", "--threads", "0"},
	} {
		cmd := makeMergeCmd()
		cmd.SetArgs(args)
		require.Error(t, cmd.Execute())
	}
}
