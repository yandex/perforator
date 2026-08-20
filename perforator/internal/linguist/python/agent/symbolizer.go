package agent

import (
	"errors"
	"fmt"

	"github.com/yandex/perforator/perforator/internal/linguist/python/agent/linetable"
	"github.com/yandex/perforator/perforator/internal/linguist/python/agent/remotemem"
	pyoffsets "github.com/yandex/perforator/perforator/internal/linguist/python/offsets"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

// SymbolSource provides name/filename symbolization for interpreter frames.
type SymbolSource interface {
	Symbolize(key *unwinder.SymbolKey) (*symbolizer.Symbol, bool)
}

// OffsetsLookup provides per-pid CPython internals offsets for line resolution.
type OffsetsLookup interface {
	OffsetsForPid(pid uint32) (*unwinder.PythonInternalsOffsets, bool)
}

// codeLinetableReader reads and validates co_linetable from a remote process.
type codeLinetableReader interface {
	ReadCodeLinetable(
		pid uint32,
		codeObjectAddr uintptr,
		expectedLinetableAddr uintptr,
		offsets remotemem.CodeObjectOffsets,
		expectedFirstlineno int32,
	) (linetable.LocationTable, error)
}

var _ codeLinetableReader = (*remotemem.Reader)(nil)

// SymbolizerConfig extends the shared symbolizer cache settings with linetable cache config.
type SymbolizerConfig struct {
	symbolizer.SymbolizerConfig `yaml:",inline"`
	LinetableCache              linetable.Config `yaml:"linetable_cache"`
}

// Symbolizer symbolizes Python frames and optionally resolves source lines.
type Symbolizer struct {
	symbols SymbolSource
	offsets OffsetsLookup
	reader  codeLinetableReader
	cache   *linetable.Cache
}

// NewSymbolizer builds a Symbolizer. A nil offsets disables line resolution.
func NewSymbolizer(symbols SymbolSource, offsets OffsetsLookup, cfg SymbolizerConfig) (*Symbolizer, error) {
	if symbols == nil {
		return nil, fmt.Errorf("python symbolizer: nil SymbolSource")
	}
	var (
		reader codeLinetableReader
		cache  *linetable.Cache
	)
	if offsets != nil {
		var err error
		cache, err = linetable.NewCache(cfg.LinetableCache)
		if err != nil {
			return nil, fmt.Errorf("python symbolizer: linetable cache: %w", err)
		}
		reader = remotemem.New()
	}
	return newSymbolizer(symbols, offsets, reader, cache), nil
}

func newSymbolizer(
	symbols SymbolSource,
	offsets OffsetsLookup,
	reader codeLinetableReader,
	cache *linetable.Cache,
) *Symbolizer {
	return &Symbolizer{
		symbols: symbols,
		offsets: offsets,
		reader:  reader,
		cache:   cache,
	}
}

// SymbolizeFrame symbolizes frame and best-effort resolves its source line.
func (s *Symbolizer) SymbolizeFrame(
	pid uint32,
	frame *unwinder.PythonFrame,
) (*symbolizer.Symbol, int32, bool) {
	if s == nil || s.symbols == nil || frame == nil {
		return nil, 0, false
	}
	sym, ok := s.symbols.Symbolize(&frame.SymbolKey)
	if !ok {
		return nil, 0, false
	}
	line, _ := s.resolveLine(pid, frame)
	return sym, line, true
}

func (s *Symbolizer) InvalidatePid(pid uint32) {
	if s == nil || s.cache == nil {
		return
	}
	s.cache.InvalidatePid(pid)
}

// Stop releases background resources owned by the symbolizer.
func (s *Symbolizer) Stop() {
	if s == nil || s.cache == nil {
		return
	}
	s.cache.Stop()
}

func (s *Symbolizer) resolveLine(pid uint32, frame *unwinder.PythonFrame) (int32, bool) {
	if s.offsets == nil || s.reader == nil || s.cache == nil {
		return 0, false
	}
	if frame.SymbolKey.Linestart == trampolineLinestart {
		return 0, false
	}
	codeObjectAddr := frame.SymbolKey.ObjectAddr
	instrPtr := frame.InstrPtr
	coLinetablePtr := frame.CoLinetablePtr
	coFirstlineno := frame.SymbolKey.Linestart
	if codeObjectAddr == 0 || instrPtr == 0 || coLinetablePtr == 0 {
		return 0, false
	}

	offsets, ok := s.offsets.OffsetsForPid(pid)
	if !ok || offsets == nil {
		return 0, false
	}

	co := offsets.PyCodeObjectOffsets
	if co.CoCodeAdaptive == pyoffsets.UnspecifiedOffset ||
		co.CoLinetable == pyoffsets.UnspecifiedOffset ||
		co.CoFirstlineno == pyoffsets.UnspecifiedOffset {
		return 0, false
	}
	bytesOff := offsets.PyBytesObjectOffsets
	if bytesOff.ObSize == pyoffsets.UnspecifiedOffset || bytesOff.ObSval == pyoffsets.UnspecifiedOffset {
		return 0, false
	}

	bytecodeOffset, ok := linetable.InstrPtrToBytecodeOffset(
		instrPtr,
		codeObjectAddr,
		uint64(co.CoCodeAdaptive),
	)
	if !ok {
		return 0, false
	}

	cacheKey := linetable.CacheKey{
		Pid:            pid,
		CodeObjectPtr:  codeObjectAddr,
		CoLinetablePtr: coLinetablePtr,
		CoFirstlineno:  coFirstlineno,
	}

	if table, hit := s.cache.Get(cacheKey); hit {
		if table.Unresolvable {
			return 0, false
		}
		return validLine(table.ResolveLine(bytecodeOffset))
	}

	table, err := s.reader.ReadCodeLinetable(
		pid,
		uintptr(codeObjectAddr),
		uintptr(coLinetablePtr),
		remotemem.CodeObjectOffsets{
			CoFirstlineno: co.CoFirstlineno,
			CoLinetable:   co.CoLinetable,
			BytesObSize:   bytesOff.ObSize,
			BytesObSval:   bytesOff.ObSval,
		},
		coFirstlineno,
	)
	if err != nil {
		// ErrCodeObjectChanged is a transient race between BPF capture and the
		// userspace read; skip the line for this sample only and retry later.
		if !errors.Is(err, remotemem.ErrCodeObjectChanged) {
			s.cache.AddTombstone(cacheKey)
		}
		return 0, false
	}

	s.cache.Add(cacheKey, table)
	return validLine(table.ResolveLine(bytecodeOffset))
}

// validLine rejects non-positive lines. CPython line numbers are 1-based, but
// the location table accumulates signed deltas, so a desynced or truncated
// table can walk the running line below zero.
func validLine(line int32, ok bool) (int32, bool) {
	if !ok || line <= 0 {
		return 0, false
	}
	return line, true
}
