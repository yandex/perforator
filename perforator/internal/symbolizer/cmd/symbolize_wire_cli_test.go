//go:build cgo

package cmd

import (
	"bytes"
	"debug/elf"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	pprof "github.com/google/pprof/profile"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protowire"

	"github.com/yandex/perforator/library/go/test/yatest"
	"github.com/yandex/perforator/perforator/pkg/xelf"
)

// Execute the registered production Cobra command, not a writer or batch helper.
func wireCLI(t *testing.T, args ...string) error {
	t.Helper()
	saved := localArgs
	changed := make(map[string]bool)
	symbolizeLocalCmd.Flags().VisitAll(func(f *pflag.Flag) {
		changed[f.Name] = f.Changed
		f.Changed = false
	})
	localArgs = symbolizeLocalArgs{
		ReuseProfileWire: symbolizeLocalCmd.Flags().Lookup("reuse-profile-wire").DefValue == "true",
	}
	defer func() {
		localArgs = saved
		symbolizeLocalCmd.Flags().VisitAll(func(f *pflag.Flag) { f.Changed = changed[f.Name] })
		rootCmd.SetArgs(nil)
	}()
	rootCmd.SetArgs(append([]string{"symbolize", "local"}, args...))
	_, err := rootCmd.ExecuteC()
	return err
}

func wireCLIFixture(t *testing.T) (string, []byte) {
	t.Helper()
	path := yatest.SourcePath("perforator/symbolizer/tests/sample_program.elf")
	buildID, err := xelf.GetBuildID(path)
	require.NoError(t, err)
	file, err := elf.Open(path)
	require.NoError(t, err)
	defer file.Close()
	symbols, err := file.Symbols()
	require.NoError(t, err)
	var load, address uint64
	for _, program := range file.Progs {
		if program.Type == elf.PT_LOAD {
			load = program.Vaddr
			break
		}
	}
	for _, symbol := range symbols {
		if symbol.Name == "main" {
			address = symbol.Value
			break
		}
	}
	require.NotZero(t, address, "fixture must have a real main symbol")
	const base = 0x10000000
	mapping := &pprof.Mapping{ID: 1, BuildID: buildID, File: "producer", Start: base, Limit: base + address - load + 4096}
	producer := &pprof.Function{ID: 1, Name: "producer", SystemName: "producer", Filename: "producer.cc"}
	locations := []*pprof.Location{
		{ID: 1, Mapping: mapping, Address: base, Line: []pprof.Line{{Function: producer, Line: 10}}},
		{ID: 2, Mapping: mapping, Address: base + address - load},
		{ID: 3, Mapping: mapping, Address: base + address - load},
	}
	p := &pprof.Profile{
		SampleType: []*pprof.ValueType{{Type: "cpu", Unit: "cycles"}},
		Mapping:    []*pprof.Mapping{mapping}, Function: []*pprof.Function{producer}, Location: locations,
		Sample:    []*pprof.Sample{{Location: locations, Value: []int64{7}, Label: map[string][]string{"key": {"value"}}}},
		TimeNanos: 12, DurationNanos: 60e9, Comments: []string{"unchanged"},
	}
	var raw bytes.Buffer
	require.NoError(t, p.WriteUncompressed(&raw))
	// A producer-only table entry makes preservation observable without hooks.
	data := append([]byte(nil), raw.Bytes()...)
	data = protowire.AppendBytes(protowire.AppendTag(data, 6, protowire.BytesType), []byte("unused-producer-string"))
	return path, data
}

func wireCLIFields(t *testing.T, raw []byte, tag protowire.Number) [][]byte {
	t.Helper()
	var fields [][]byte
	for len(raw) > 0 {
		number, _, n := protowire.ConsumeField(raw)
		require.Positive(t, n)
		if number == tag {
			fields = append(fields, raw[:n])
		}
		raw = raw[n:]
	}
	return fields
}

func TestWireCLINativeOutputMatchesDefault(t *testing.T) {
	binaryPath, raw := wireCLIFixture(t)
	original := wireCLIFields(t, raw, 4)[1]
	_, _, n := protowire.ConsumeTag(original)
	body, n := protowire.ConsumeBytes(original[n:])
	require.Positive(t, n)
	producer := protowire.AppendBytes(protowire.AppendTag(append([]byte(nil), body...), 100, protowire.BytesType), []byte("producer-extension"))
	// An explicit default after an unknown field must not be normalized away.
	producer = protowire.AppendVarint(protowire.AppendTag(producer, 5, protowire.VarintType), 0)
	replacement := protowire.AppendBytes(protowire.AppendTag(nil, 4, protowire.BytesType), producer)
	raw = bytes.Replace(raw, original, replacement, 1)

	dir := t.TempDir()
	input := filepath.Join(dir, "input.pprof")
	require.NoError(t, os.WriteFile(input, raw, 0600))
	var profiles [][]byte
	for _, reuseWire := range []bool{false, true} {
		outputDir := filepath.Join(dir, "default")
		if reuseWire {
			outputDir = filepath.Join(dir, "wire")
		}
		args := []string{"--profiles", input, "--output-dir", outputDir, "--binary", binaryPath}
		if reuseWire {
			args = append(args, "--reuse-profile-wire")
		}
		require.NoError(t, wireCLI(t, args...))
		output, err := os.ReadFile(filepath.Join(outputDir, filepath.Base(input)))
		require.NoError(t, err)
		p, err := pprof.ParseData(output)
		require.NoError(t, err)
		require.NoError(t, p.CheckValid())
		require.Equal(t, "producer", p.Function[0].Name)
		require.NotEmpty(t, p.Location[1].Line)
		require.Equal(t, "main", p.Location[1].Line[0].Function.SystemName)
		require.Positive(t, p.Location[1].Line[0].Line)
		require.NotEqual(t, p.Location[1].Line[0].Function.ID, p.Location[2].Line[0].Function.ID)
		public, err := json.Marshal(p)
		require.NoError(t, err)
		profiles = append(profiles, public)
		if reuseWire {
			require.Equal(t, wireCLIFields(t, raw, 2), wireCLIFields(t, output, 2))
			require.Contains(t, string(output), "unused-producer-string")
			location := wireCLIFields(t, output, 4)[1]
			_, _, n := protowire.ConsumeTag(location)
			body, n := protowire.ConsumeBytes(location[n:])
			require.Positive(t, n)
			require.True(t, bytes.HasPrefix(body, producer), "producer Location bytes must remain an exact prefix")
		} else {
			require.NotContains(t, string(output), "unused-producer-string")
		}
	}
	require.Equal(t, profiles[0], profiles[1], "all public tables/order/IDs/references and metadata must match")
}
