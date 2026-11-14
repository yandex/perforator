package profiler

import (
	"encoding/binary"
	"testing"
	"unsafe"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/nativepipeline"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/nativepipeline/runtimewriter"
	lua "github.com/yandex/perforator/perforator/internal/linguist/lua/agent"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

func TestResolveLuaSectionWithMetadata(t *testing.T) {
	frames := []unwinder.LuaFrame{
		{Type: unwinder.LuaFrameTypeVmInfo},
		{Type: unwinder.LuaFrameTypeInvalid},
	}
	binary.LittleEndian.PutUint64(frames[0].Value.UnionBuf[:], 0x1000)
	binary.LittleEndian.PutUint64(frames[0].Value.UnionBuf[8:], 0x2000)
	section := nativepipeline.LanguageSection{
		Language:  nativepipeline.LanguageLua,
		FrameSize: uint16(unsafe.Sizeof(frames[0])),
		Data:      unsafe.Slice((*byte)(unsafe.Pointer(&frames[0])), len(frames)*int(unsafe.Sizeof(frames[0]))),
	}
	writer, err := runtimewriter.New(3, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	processor := lua.NewStackProcessor(nil, nop.Registry{})
	resolved := 0
	resolve := func(key linux.ProcessKey, frame, caller *unwinder.LuaFrame, output runtimewriter.Frame) error {
		if frame != &frames[resolved] {
			t.Fatal("header changed frame order")
		}
		var wantCaller *unwinder.LuaFrame
		if resolved+1 < len(frames) {
			wantCaller = &frames[resolved+1]
		}
		if caller != wantCaller {
			t.Fatal("header changed caller link")
		}
		resolved++
		return processor.ResolveFrame(key, frame, caller, output)
	}
	if err := resolveLuaSection(resolve, linux.ProcessKey{Pid: 1}, section, writer); err != nil {
		t.Fatal(err)
	}
	if resolved != len(frames) {
		t.Fatalf("resolved %d frames, want %d", resolved, len(frames))
	}
	// Also validates that the final frame's inline chain is complete.
	if _, err := writer.AddFrame(0, 0); err != nil {
		t.Fatal(err)
	}
}
