package agent

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	python_models "github.com/yandex/perforator/perforator/internal/linguist/python/models"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
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
	key := unwinder.SymbolKey{ObjectAddr: 0xabc, Pid: 1, Linestart: 10}
	source := &stubSymbolSource{
		symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{
			key: {Name: "foo", FileName: "busyloop.py"},
		},
	}
	proc := newTestStackProcessor(t, source)

	builder := profile.NewBuilder().AddSampleType("cpu", "cycles").Add(1).AddValue(1)

	proc.Process(builder, singleFrameStack(0xabc, 10, 0x1000), 1)

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
	key := unwinder.SymbolKey{ObjectAddr: 0xabc, Pid: 1, Linestart: 10}
	source := &stubSymbolSource{
		symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{
			key: {Name: "foo", FileName: "busyloop.py"},
		},
	}
	reader := &stubLinetableReader{table: testLocationTableWithLineDelta()}
	sym := newTestSymbolizer(t, source, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)
	proc := NewStackProcessor(sym, nop.Registry{})

	builder := profile.NewBuilder().AddSampleType("cpu", "cycles").Add(1).AddValue(1)
	proc.Process(builder, singleFrameStack(0xabc, 10, 0xabc+0x80+2), 1)

	p := builder.Finish().Finish()
	require.Len(t, p.Sample, 1)
	loc := p.Sample[0].Location[0]
	require.Len(t, loc.Line, 1)
	require.Equal(t, int64(12), loc.Line[0].Line)
	require.Equal(t, int64(10), loc.Line[0].Function.StartLine)
}

func TestStackProcessor_Unsymbolized(t *testing.T) {
	proc := newTestStackProcessor(t, &stubSymbolSource{symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{}})

	builder := profile.NewBuilder().AddSampleType("cpu", "cycles").Add(1).AddValue(1)

	proc.Process(builder, singleFrameStack(0xabc, 10, 0x1000), 1)

	p := builder.Finish().Finish()
	require.Len(t, p.Sample, 1)
	loc := p.Sample[0].Location[0]
	require.Len(t, loc.Line, 1)
	require.Equal(t, models.UnsymbolizedInterpreterLocation, loc.Line[0].Function.Name)
	require.Equal(t, int64(10), loc.Line[0].Function.StartLine)
}

func TestStackProcessor_Trampoline(t *testing.T) {
	proc := newTestStackProcessor(t, &stubSymbolSource{symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{}})

	builder := profile.NewBuilder().AddSampleType("cpu", "cycles").Add(1).AddValue(1)

	proc.Process(builder, singleFrameStack(0xabc, -1, 0x1000), 1)

	p := builder.Finish().Finish()
	require.Len(t, p.Sample, 1)
	loc := p.Sample[0].Location[0]
	require.Len(t, loc.Line, 1)
	require.Equal(t, python_models.PythonTrampolineFrame, loc.Line[0].Function.Name)
	require.Equal(t, int64(0), loc.Line[0].Line)
}
