package agent

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/internal/linguist/python/agent/linetable"
	"github.com/yandex/perforator/perforator/internal/linguist/python/agent/remotemem"
	pyoffsets "github.com/yandex/perforator/perforator/internal/linguist/python/offsets"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

type stubOffsetsLookup struct {
	offsets *unwinder.PythonInternalsOffsets
	ok      bool
}

func (s *stubOffsetsLookup) OffsetsForPid(uint32) (*unwinder.PythonInternalsOffsets, bool) {
	return s.offsets, s.ok
}

type stubLinetableReader struct {
	table                     linetable.LocationTable
	err                       error
	calls                     int
	lastExpectedLinetableAddr uintptr
}

func (s *stubLinetableReader) ReadCodeLinetable(
	_ uint32,
	_ uintptr,
	expectedLinetableAddr uintptr,
	_ remotemem.CodeObjectOffsets,
	_ int32,
) (linetable.LocationTable, error) {
	s.calls++
	s.lastExpectedLinetableAddr = expectedLinetableAddr
	if s.err != nil {
		return linetable.LocationTable{}, s.err
	}
	return s.table, nil
}

type stubSymbolSource struct {
	symbols map[unwinder.SymbolKey]*symbolizer.Symbol
}

func (s *stubSymbolSource) Symbolize(key *unwinder.SymbolKey) (*symbolizer.Symbol, bool) {
	sym, ok := s.symbols[*key]
	return sym, ok
}

func testOffsets() *unwinder.PythonInternalsOffsets {
	var offsets unwinder.PythonInternalsOffsets
	offsets.PyCodeObjectOffsets.CoCodeAdaptive = 0x80
	offsets.PyCodeObjectOffsets.CoLinetable = 0x40
	offsets.PyCodeObjectOffsets.CoFirstlineno = 0x10
	offsets.PyBytesObjectOffsets.ObSize = 0
	offsets.PyBytesObjectOffsets.ObSval = 8
	return &offsets
}

func testLocationTable() linetable.LocationTable {
	// SHORT entry covering bytecode bytes [0, 6) at firstlineno.
	return linetable.LocationTable{
		FirstLineno: 10,
		Raw:         []byte{0x80 | (0 << 3) | 2, 0x00}, // code=0, length=3 code units
	}
}

func testLocationTableWithLineDelta() linetable.LocationTable {
	return linetable.LocationTable{
		FirstLineno: 10,
		Raw: []byte{
			0x80 | (0 << 3), 0x00, // SHORT: bytecode bytes [0, 2), line 10
			0x80 | (12 << 3), 0x00, 0x00, // ONE_LINE_2: bytecode bytes [2, 4), line 12
		},
	}
}

func testFrame(objectAddr uint64, linestart int32, instrPtr uint64, coLinetablePtr uint64) *unwinder.PythonFrame {
	return &unwinder.PythonFrame{
		SymbolKey: unwinder.SymbolKey{
			ObjectAddr: objectAddr,
			Pid:        1,
			Linestart:  linestart,
		},
		InstrPtr:       instrPtr,
		CoLinetablePtr: coLinetablePtr,
	}
}

func newTestSymbolizer(
	t *testing.T,
	symbols SymbolSource,
	offsets OffsetsLookup,
	reader *stubLinetableReader,
) *Symbolizer {
	t.Helper()
	cache, err := linetable.NewCache(linetable.Config{})
	require.NoError(t, err)
	t.Cleanup(cache.Stop)
	return newSymbolizer(symbols, offsets, reader, cache)
}

func TestResolveLine_OK(t *testing.T) {
	reader := &stubLinetableReader{table: testLocationTable()}
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	const (
		pid            = uint32(42)
		codeObjectAddr = uint64(0x1000)
		coCodeAdaptive = uint64(0x80)
		coLinetablePtr = uint64(0x2000)
		firstlineno    = int32(10)
	)
	instrPtr := codeObjectAddr + coCodeAdaptive + 2 // bytecode offset 2 → line 10
	frame := testFrame(codeObjectAddr, firstlineno, instrPtr, coLinetablePtr)

	line, ok := sym.resolveLine(pid, frame)
	require.True(t, ok)
	require.Equal(t, int32(10), line)
	require.Equal(t, 1, reader.calls)
	require.Equal(t, uintptr(coLinetablePtr), reader.lastExpectedLinetableAddr)

	line, ok = sym.resolveLine(pid, frame)
	require.True(t, ok)
	require.Equal(t, int32(10), line)
	require.Equal(t, 1, reader.calls)
}

func TestResolveLine_UsesLinetableLineDelta(t *testing.T) {
	reader := &stubLinetableReader{table: testLocationTableWithLineDelta()}
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)
	frame := testFrame(0x1000, 10, 0x1082, 0x2000)

	line, ok := sym.resolveLine(42, frame)
	require.True(t, ok)
	require.Equal(t, int32(12), line)
	require.Equal(t, 1, reader.calls)

	line, ok = sym.resolveLine(42, frame)
	require.True(t, ok)
	require.Equal(t, int32(12), line)
	require.Equal(t, 1, reader.calls, "cache hit must preserve the resolved line")
}

func TestInvalidatePid_ForcesReread(t *testing.T) {
	reader := &stubLinetableReader{table: testLocationTable()}
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	const (
		pid            = uint32(42)
		codeObjectAddr = uint64(0x1000)
		coCodeAdaptive = uint64(0x80)
		firstlineno    = int32(10)
	)
	frame := testFrame(codeObjectAddr, firstlineno, codeObjectAddr+coCodeAdaptive+2, 0x2000)

	_, ok := sym.resolveLine(pid, frame)
	require.True(t, ok)
	require.Equal(t, 1, reader.calls)

	sym.InvalidatePid(pid)

	_, ok = sym.resolveLine(pid, frame)
	require.True(t, ok)
	require.Equal(t, 2, reader.calls, "cache must miss after InvalidatePid")
}

func TestResolveLine_MissingOffsets(t *testing.T) {
	reader := &stubLinetableReader{table: testLocationTable()}
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{ok: false}, reader)

	line, ok := sym.resolveLine(1, testFrame(0x1000, 10, 0x1080, 0x2000))
	require.False(t, ok)
	require.Equal(t, int32(0), line)
	require.Equal(t, 0, reader.calls)
}

func TestResolveLine_MissingCoFirstlinenoOffset(t *testing.T) {
	reader := &stubLinetableReader{table: testLocationTable()}
	offsets := testOffsets()
	offsets.PyCodeObjectOffsets.CoFirstlineno = pyoffsets.UnspecifiedOffset
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{offsets: offsets, ok: true}, reader)

	line, ok := sym.resolveLine(1, testFrame(0x1000, 10, 0x1080, 0x2000))
	require.False(t, ok)
	require.Equal(t, int32(0), line)
	require.Equal(t, 0, reader.calls)
}

func TestResolveLine_RemoteErrorTombstone(t *testing.T) {
	reader := &stubLinetableReader{err: errors.New("boom")}
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	frame := testFrame(0x1000, 10, 0x1082, 0x2000)
	line, ok := sym.resolveLine(1, frame)
	require.False(t, ok)
	require.Equal(t, int32(0), line)
	require.Equal(t, 1, reader.calls)

	line, ok = sym.resolveLine(1, frame)
	require.False(t, ok)
	require.Equal(t, int32(0), line)
	require.Equal(t, 1, reader.calls)
}

func TestResolveLine_CodeObjectChangedDoesNotTombstone(t *testing.T) {
	reader := &stubLinetableReader{err: remotemem.ErrCodeObjectChanged}
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	frame := testFrame(0x1000, 10, 0x1082, 0x2000)
	line, ok := sym.resolveLine(1, frame)
	require.False(t, ok)
	require.Equal(t, int32(0), line)
	require.Equal(t, 1, reader.calls)

	reader.err = nil
	reader.table = testLocationTable()

	line, ok = sym.resolveLine(1, frame)
	require.True(t, ok)
	require.Equal(t, int32(10), line)
	require.Equal(t, 2, reader.calls, "must retry remote read after ErrCodeObjectChanged")
}

func TestResolveLine_RejectsNonPositiveLine(t *testing.T) {
	table := testLocationTable()
	table.FirstLineno = 0
	reader := &stubLinetableReader{table: table}
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	line, ok := sym.resolveLine(1, testFrame(0x1000, 0, 0x1082, 0x2000))
	require.False(t, ok)
	require.Equal(t, int32(0), line)
}

func TestResolveLine_ZeroPointers(t *testing.T) {
	reader := &stubLinetableReader{table: testLocationTable()}
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	_, ok := sym.resolveLine(1, testFrame(0, 10, 0x1080, 0x2000))
	require.False(t, ok)
	_, ok = sym.resolveLine(1, testFrame(0x1000, 10, 0, 0x2000))
	require.False(t, ok)
	_, ok = sym.resolveLine(1, testFrame(0x1000, 10, 0x1080, 0))
	require.False(t, ok)
	require.Equal(t, 0, reader.calls)
}

func TestResolveLine_ChangedLinetablePointerMissesCache(t *testing.T) {
	reader := &stubLinetableReader{table: testLocationTable()}
	sym := newTestSymbolizer(t, &stubSymbolSource{}, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	_, ok := sym.resolveLine(1, testFrame(0x1000, 10, 0x1082, 0x2000))
	require.True(t, ok)
	_, ok = sym.resolveLine(1, testFrame(0x1000, 10, 0x1082, 0x3000))
	require.True(t, ok)
	require.Equal(t, 2, reader.calls)
}

func TestSymbolizeFrame_SymbolAndLine(t *testing.T) {
	key := unwinder.SymbolKey{ObjectAddr: 0x1000, Pid: 1, Linestart: 10}
	source := &stubSymbolSource{
		symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{
			key: {Name: "foo", FileName: "busyloop.py"},
		},
	}
	reader := &stubLinetableReader{table: testLocationTable()}
	sym := newTestSymbolizer(t, source, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	instrPtr := uint64(0x1000) + 0x80 + 2
	got, line, ok := sym.SymbolizeFrame(42, testFrame(0x1000, 10, instrPtr, 0x2000))
	require.True(t, ok)
	require.Equal(t, "foo", got.Name)
	require.Equal(t, "busyloop.py", got.FileName)
	require.Equal(t, int32(10), line)
	require.Equal(t, 1, reader.calls)
}

func TestSymbolizeFrame_Unsymbolized(t *testing.T) {
	reader := &stubLinetableReader{table: testLocationTable()}
	sym := newTestSymbolizer(t, &stubSymbolSource{symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{}}, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	got, line, ok := sym.SymbolizeFrame(1, testFrame(0x1000, 10, 0x1082, 0x2000))
	require.False(t, ok)
	require.Nil(t, got)
	require.Equal(t, int32(0), line)
	require.Equal(t, 0, reader.calls)
}

func TestSymbolizeFrame_LineResolutionDisabled(t *testing.T) {
	key := unwinder.SymbolKey{ObjectAddr: 0x1000, Pid: 1, Linestart: 10}
	source := &stubSymbolSource{
		symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{
			key: {Name: "foo", FileName: "busyloop.py"},
		},
	}
	// offsets == nil disables line resolution (feature flag off).
	sym := newSymbolizer(source, nil, nil, nil)

	got, line, ok := sym.SymbolizeFrame(1, testFrame(0x1000, 10, 0x1082, 0x2000))
	require.True(t, ok)
	require.Equal(t, "foo", got.Name)
	require.Equal(t, int32(0), line)
}

func TestSymbolizeFrame_TrampolineSkipsLine(t *testing.T) {
	key := unwinder.SymbolKey{ObjectAddr: 0x1000, Pid: 1, Linestart: -1}
	source := &stubSymbolSource{
		symbols: map[unwinder.SymbolKey]*symbolizer.Symbol{
			key: {Name: "trampoline", FileName: ""},
		},
	}
	reader := &stubLinetableReader{table: testLocationTable()}
	sym := newTestSymbolizer(t, source, &stubOffsetsLookup{offsets: testOffsets(), ok: true}, reader)

	got, line, ok := sym.SymbolizeFrame(1, testFrame(0x1000, -1, 0x1082, 0x2000))
	require.True(t, ok)
	require.Equal(t, "trampoline", got.Name)
	require.Equal(t, int32(0), line)
	require.Equal(t, 0, reader.calls)
}
