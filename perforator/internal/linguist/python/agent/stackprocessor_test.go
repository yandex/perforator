package agent

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	python_models "github.com/yandex/perforator/perforator/internal/linguist/python/models"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

func newTestStackProcessor(t *testing.T, source SymbolSource) *StackProcessor {
	t.Helper()
	sym, err := NewSymbolizer(source, nil, SymbolizerConfig{})
	require.NoError(t, err)
	return NewStackProcessor(sym, nop.Registry{})
}

func singleFrameStack(objectAddr uint64, linestart int32, instrPtr uint64) *unwinder.PythonStack {
	stack := &unwinder.PythonStack{Len: 1}
	stack.Frames[0] = *testFrame(objectAddr, linestart, instrPtr, 0x2000)
	return stack
}

func TestStackProcessor_SetsNameFilenameStartLine(t *testing.T) {
	key := unwinder.SymbolKey{ObjectAddr: 0xabc, Linestart: 10}
	source := &stubSymbolSource{
		symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{
			key: {Name: "foo", FileName: "busyloop.py"},
		},
	}
	proc := newTestStackProcessor(t, source)

	builder := profile.NewBuilder().AddSampleType("cpu", "cycles").Add(linux.ProcessKey{Pid: 1}).AddValue(1)

	proc.Process(builder, singleFrameStack(0xabc, 10, 0x1000), linux.ProcessKey{Pid: 1, ProcessStartTime: 123})
	require.Equal(t, models.Language(unwinder.LanguagePython), source.language)
	require.Equal(t, linux.ProcessKey{Pid: 1, ProcessStartTime: 123}, source.process)

	p := builder.Finish().Finish()
	require.Len(t, p.Sample, 1)
	require.Len(t, p.Sample[0].Location, 1)
	loc := p.Sample[0].Location[0]
	require.Len(t, loc.Line, 1)
	require.Equal(t, int64(0), loc.Line[0].Line) // line resolution disabled
	require.Equal(t, "foo", loc.Line[0].Function.Name)
	require.Equal(t, "busyloop.py", loc.Line[0].Function.Filename)
	require.Equal(t, int64(10), loc.Line[0].Function.StartLine)
}

func TestStackProcessor_SetsResolvedLine(t *testing.T) {
	key := unwinder.SymbolKey{ObjectAddr: 0xabc, Linestart: 10}
	source := &stubSymbolSource{
		symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{
			key: {Name: "foo", FileName: "busyloop.py"},
		},
	}
	reader := &stubLinetableReader{table: testLocationTableWithLineDelta()}
	sym := newTestSymbolizer(t, source, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)
	proc := NewStackProcessor(sym, nop.Registry{})

	builder := profile.NewBuilder().AddSampleType("cpu", "cycles").Add(linux.ProcessKey{Pid: 1}).AddValue(1)
	proc.Process(builder, singleFrameStack(0xabc, 10, 0xabc+0x80+2), linux.ProcessKey{Pid: 1})

	p := builder.Finish().Finish()
	require.Len(t, p.Sample, 1)
	loc := p.Sample[0].Location[0]
	require.Len(t, loc.Line, 1)
	require.Equal(t, int64(12), loc.Line[0].Line)
	require.Equal(t, int64(10), loc.Line[0].Function.StartLine)
}

func TestStackProcessor_Unsymbolized(t *testing.T) {
	proc := newTestStackProcessor(t, &stubSymbolSource{symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{}})

	builder := profile.NewBuilder().AddSampleType("cpu", "cycles").Add(linux.ProcessKey{Pid: 1}).AddValue(1)

	proc.Process(builder, singleFrameStack(0xabc, 10, 0x1000), linux.ProcessKey{Pid: 1})

	p := builder.Finish().Finish()
	require.Len(t, p.Sample, 1)
	loc := p.Sample[0].Location[0]
	require.Len(t, loc.Line, 1)
	require.Equal(t, models.UnsymbolizedInterpreterLocation, loc.Line[0].Function.Name)
	require.Equal(t, int64(10), loc.Line[0].Function.StartLine)
}

func TestStackProcessor_Trampoline(t *testing.T) {
	proc := newTestStackProcessor(t, &stubSymbolSource{symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{}})

	builder := profile.NewBuilder().AddSampleType("cpu", "cycles").Add(linux.ProcessKey{Pid: 1}).AddValue(1)

	proc.Process(builder, singleFrameStack(0xabc, -1, 0x1000), linux.ProcessKey{Pid: 1})

	p := builder.Finish().Finish()
	require.Len(t, p.Sample, 1)
	loc := p.Sample[0].Location[0]
	require.Len(t, loc.Line, 1)
	require.Equal(t, python_models.PythonTrampolineFrame, loc.Line[0].Function.Name)
	require.Equal(t, int64(0), loc.Line[0].Line)
}

func TestStackProcessor_LocationIdentity(t *testing.T) {
	for _, kind := range []string{"symbolized", "unsymbolized", "trampoline"} {
		t.Run(kind, func(t *testing.T) {
			key := unwinder.SymbolKey{ObjectAddr: 0xabc, Linestart: 10}
			if kind == "trampoline" {
				key.Linestart = trampolineLinestart
			}
			source := &stubSymbolSource{symbols: make(map[unwinder.SymbolKey]*symbolizer.Symbol)}
			proc := newTestStackProcessor(t, source)
			builder := profile.NewBuilder().AddSampleType("cpu", "cycles")

			// An earlier PHP frame with the same address/lines must not mask Python.
			phpSample := builder.Add(linux.ProcessKey{Pid: 1, ProcessStartTime: 100}).AddValue(1)
			phpLoc := phpSample.AddInterpreterLocation(&profile.InterpreterLocationKey{
				ObjectAddress: key.ObjectAddr,
				Linestart:     key.Linestart,
				Language:      models.Language(unwinder.LanguagePhp),
			})
			phpLoc.SetMapping().SetPath(profile.PHPSpecialMapping).Finish()
			phpLoc.AddFrame().SetName("php-function").Finish()
			phpLoc.Finish()
			phpSample.Finish()

			startTimes := []uint64{100, 200, 100}
			for _, startTime := range startTimes {
				if kind == "symbolized" {
					source.symbols[key] = &symbolizer.Symbol{
						Name:     fmt.Sprintf("process-%d", startTime),
						FileName: fmt.Sprintf("process-%d.py", startTime),
					}
				}
				sample := builder.Add(linux.ProcessKey{Pid: 1, ProcessStartTime: startTime}).AddValue(1)
				proc.Process(sample, singleFrameStack(key.ObjectAddr, key.Linestart, 0x1000), linux.ProcessKey{Pid: 1, ProcessStartTime: startTime})
				sample.Finish()
			}

			p := builder.FinishRaw()
			require.Len(t, p.Sample, 4)
			require.Len(t, p.Location, 3)
			require.Equal(t, profile.PHPSpecialMapping, p.Sample[0].Location[0].Mapping.File)
			require.Equal(t, "php-function", p.Sample[0].Location[0].Line[0].Function.Name)
			for i, startTime := range startTimes {
				loc := p.Sample[i+1].Location[0]
				require.Equal(t, profile.PythonSpecialMapping, loc.Mapping.File)
				require.Len(t, loc.Line, 1)
				switch kind {
				case "symbolized":
					require.Equal(t, fmt.Sprintf("process-%d", startTime), loc.Line[0].Function.Name)
					require.Equal(t, fmt.Sprintf("process-%d.py", startTime), loc.Line[0].Function.Filename)
				case "unsymbolized":
					require.Equal(t, models.UnsymbolizedInterpreterLocation, loc.Line[0].Function.Name)
				case "trampoline":
					require.Equal(t, python_models.PythonTrampolineFrame, loc.Line[0].Function.Name)
				}
			}
			require.NotEqual(t, p.Sample[1].Location[0].ID, p.Sample[2].Location[0].ID)
			require.Same(t, p.Sample[1].Location[0], p.Sample[3].Location[0])
		})
	}
}
