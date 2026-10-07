package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	pprof "github.com/google/pprof/profile"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestLocalProfileArguments(t *testing.T) {
	for _, tc := range []struct {
		name       string
		args       []string
		options    symbolizeLocalArgs
		outputFlag bool
		valid      bool
	}{
		{name: "legacy", args: []string{"one.pprof"}, valid: true},
		{name: "missing input"},
		{name: "multiple positional inputs", args: []string{"one", "two"}},
		{name: "batch", options: symbolizeLocalArgs{ProfilePaths: []string{"one", "two"}, OutputDir: "out"}, valid: true},
		{name: "mixed inputs", args: []string{"one"}, options: symbolizeLocalArgs{ProfilePaths: []string{"two"}, OutputDir: "out"}},
		{name: "missing directory", options: symbolizeLocalArgs{ProfilePaths: []string{"one"}}},
		{name: "mixed outputs", options: symbolizeLocalArgs{ProfilePaths: []string{"one"}, OutputDir: "out"}, outputFlag: true},
		{name: "directory without batch", args: []string{"one"}, options: symbolizeLocalArgs{OutputDir: "out"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "local"}
			cmd.Flags().String("output", "symbolized_profile.pprof", "")
			if tc.outputFlag {
				require.NoError(t, cmd.Flags().Set("output", "out.pprof"))
			}
			err := validateLocalArgs(cmd, tc.args, tc.options)
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestLocalProfilesRejectReportOverwritingInput(t *testing.T) {
	dir := t.TempDir()
	input := filepath.Join(dir, "input.pprof")
	require.NoError(t, os.WriteFile(input, []byte("original"), 0600))
	err := symbolizeLocalBatch(context.Background(), symbolizeLocalArgs{
		ProfilePaths: []string{input}, OutputDir: filepath.Join(dir, "out"), ReportPath: input,
	})
	require.ErrorContains(t, err, "report overlaps")
	data, err := os.ReadFile(input)
	require.NoError(t, err)
	require.Equal(t, "original", string(data))
}

func TestLocalBatchKeepsMinutesAndContinuesAfterFailure(t *testing.T) {
	dir := t.TempDir()
	jobs := []localBatchJob{
		{Input: filepath.Join(dir, "first"), Output: filepath.Join(dir, "first.out")},
		{Input: filepath.Join(dir, "bad"), Output: filepath.Join(dir, "bad.out")},
		{Input: filepath.Join(dir, "last"), Output: filepath.Join(dir, "last.out")},
	}
	for i, job := range jobs {
		p := &pprof.Profile{SampleType: []*pprof.ValueType{{Type: "cpu", Unit: "cycles"}},
			Sample: []*pprof.Sample{{Value: []int64{int64(i + 1)}}}, TimeNanos: int64(i+1) * 60e9, DurationNanos: 60e9}
		var data bytes.Buffer
		require.NoError(t, p.WriteUncompressed(&data))
		require.NoError(t, os.WriteFile(job.Input, data.Bytes(), 0600))
	}
	var report bytes.Buffer
	calls := 0
	err := runLocalBatch(context.Background(), jobs, &report, func(p *pprof.Profile) error {
		calls++
		if p.TimeNanos == 120e9 {
			return errors.New("bad binary")
		}
		p.Comments = append(p.Comments, "processed")
		return nil
	})
	require.ErrorContains(t, err, "1 of 3 profiles failed")
	require.ErrorContains(t, err, jobs[1].Input)
	require.ErrorContains(t, err, "bad binary")
	require.Equal(t, 3, calls)
	decoder := json.NewDecoder(&report)
	for i, job := range jobs {
		var result localBatchResult
		require.NoError(t, decoder.Decode(&result))
		require.Equal(t, job.Input, result.Input)
		if i == 1 {
			require.Equal(t, "bad binary", result.Error)
			_, err := os.Stat(job.Output)
			require.True(t, os.IsNotExist(err))
			continue
		}
		data, err := os.ReadFile(job.Output)
		require.NoError(t, err)
		p, err := pprof.ParseData(data)
		require.NoError(t, err)
		require.Equal(t, int64(i+1)*60e9, p.TimeNanos)
		require.Equal(t, int64(60e9), p.DurationNanos)
		require.Equal(t, []int64{int64(i + 1)}, p.Sample[0].Value)
		require.Equal(t, []string{"processed"}, p.Comments)
	}
}

func TestLocalBatchRejectsOverlappingOutputsBeforeProcessing(t *testing.T) {
	for _, jobs := range [][]localBatchJob{
		{{Input: "one", Output: "two"}, {Input: "two", Output: "three"}},
		{{Input: "one", Output: "out"}, {Input: "two", Output: "out"}},
	} {
		require.ErrorContains(t, runLocalBatch(context.Background(), jobs, &bytes.Buffer{}, func(*pprof.Profile) error {
			t.Fatal("must validate before processing")
			return nil
		}), "overlaps")
	}
}

func TestLocalProfilesRejectOutputAliases(t *testing.T) {
	for _, kind := range []string{"directory symlink", "file symlink", "hardlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			input := filepath.Join(dir, "input.pprof")
			outDir := filepath.Join(dir, "out")
			require.NoError(t, os.WriteFile(input, []byte("original"), 0600))
			if kind == "directory symlink" {
				require.NoError(t, os.Symlink(dir, outDir))
			} else {
				require.NoError(t, os.Mkdir(outDir, 0700))
				link := os.Symlink
				if kind == "hardlink" {
					link = os.Link
				}
				require.NoError(t, link(input, filepath.Join(outDir, "input.pprof")))
			}
			err := symbolizeLocalBatch(context.Background(), symbolizeLocalArgs{
				ProfilePaths: []string{input}, OutputDir: outDir,
			})
			require.ErrorContains(t, err, "batch output overlaps")
			data, err := os.ReadFile(input)
			require.NoError(t, err)
			require.Equal(t, "original", string(data))
		})
	}
}

func TestLocalProfilesRejectReportAliases(t *testing.T) {
	for _, kind := range []string{"symlink", "hardlink"} {
		for _, target := range []string{"input", "output"} {
			t.Run(kind+"/"+target, func(t *testing.T) {
				dir := t.TempDir()
				input := filepath.Join(dir, "input.pprof")
				outDir := filepath.Join(dir, "out")
				output := filepath.Join(outDir, "input.pprof")
				require.NoError(t, os.Mkdir(outDir, 0700))
				require.NoError(t, os.WriteFile(input, []byte("original input"), 0600))
				require.NoError(t, os.WriteFile(output, []byte("original output"), 0600))
				link := os.Symlink
				if kind == "hardlink" {
					link = os.Link
				}
				alias := input
				if target == "output" {
					alias = output
				}
				report := filepath.Join(dir, "report.jsonl")
				require.NoError(t, link(alias, report))
				err := symbolizeLocalBatch(context.Background(), symbolizeLocalArgs{
					ProfilePaths: []string{input}, OutputDir: outDir, ReportPath: report,
				})
				require.ErrorContains(t, err, "report overlaps")
				for path, expected := range map[string]string{input: "original input", output: "original output"} {
					data, err := os.ReadFile(path)
					require.NoError(t, err)
					require.Equal(t, expected, string(data))
				}
			})
		}
	}
}

func TestLocalBatchResolvesParentsOfNewOutputs(t *testing.T) {
	dir := t.TempDir()
	realDir := filepath.Join(dir, "real")
	aliasDir := filepath.Join(dir, "alias")
	require.NoError(t, os.Mkdir(realDir, 0700))
	require.NoError(t, os.Symlink(realDir, aliasDir))
	jobs := []localBatchJob{
		{Input: filepath.Join(dir, "one"), Output: filepath.Join(realDir, "new", "profile")},
		{Input: filepath.Join(dir, "two"), Output: filepath.Join(aliasDir, "new", "profile")},
	}
	require.ErrorContains(t, validateLocalBatch(jobs, ""), "batch output overlaps")
	require.ErrorContains(t, validateLocalBatch(jobs[:1], jobs[1].Output), "report overlaps")
	jobs[1].Output = filepath.Join(aliasDir, "new", "other")
	require.NoError(t, validateLocalBatch(jobs, filepath.Join(aliasDir, "new", "report")))
	require.NoDirExists(t, filepath.Join(realDir, "new"))
}

func TestLocalBatchRejectsLinkedOutputsAndLaterInputs(t *testing.T) {
	for _, laterInput := range []bool{false, true} {
		dir := t.TempDir()
		original := filepath.Join(dir, "original")
		alias := filepath.Join(dir, "alias")
		require.NoError(t, os.WriteFile(original, []byte("original"), 0600))
		require.NoError(t, os.Link(original, alias))
		jobs := []localBatchJob{
			{Input: filepath.Join(dir, "one"), Output: alias},
			{Input: filepath.Join(dir, "two"), Output: original},
		}
		if laterInput {
			jobs[1] = localBatchJob{Input: original, Output: filepath.Join(dir, "out")}
		}
		require.ErrorContains(t, runLocalBatch(context.Background(), jobs, io.Discard, func(*pprof.Profile) error {
			t.Fatal("must reject aliases before processing")
			return nil
		}), "batch output overlaps")
		data, err := os.ReadFile(original)
		require.NoError(t, err)
		require.Equal(t, "original", string(data))
	}
}

func TestLocalBatchReportsFailuresWithoutJSONL(t *testing.T) {
	dir := t.TempDir()
	jobs := []localBatchJob{
		{Input: filepath.Join(dir, "missing-one"), Output: filepath.Join(dir, "one.out")},
		{Input: filepath.Join(dir, "missing-two"), Output: filepath.Join(dir, "two.out")},
	}
	err := runLocalBatch(context.Background(), jobs, io.Discard, func(*pprof.Profile) error {
		t.Fatal("missing profiles must not reach symbolizer")
		return nil
	})
	require.ErrorContains(t, err, "2 of 2 profiles failed")
	require.ErrorIs(t, err, os.ErrNotExist)
	for _, job := range jobs {
		require.ErrorContains(t, err, job.Input)
		require.ErrorContains(t, err, job.Output)
	}
}
