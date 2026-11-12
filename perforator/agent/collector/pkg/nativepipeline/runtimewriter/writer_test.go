package runtimewriter

import (
	"strings"
	"testing"
	"unsafe"
)

func TestInlineFunctionMetadata(t *testing.T) {
	w, err := New(1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	frame, err := w.AddFrame(0x1234, 2)
	if err != nil {
		t.Fatal(err)
	}
	systemName := strings.Repeat("system", 1000)
	if err := frame.AddInlineFunction("function", "source", 2, 8, systemName, 1<<40); err != nil {
		t.Fatal(err)
	}
	if err := frame.AddInlineFunction("plain", "", -1, -2, "", 0); err != nil {
		t.Fatal(err)
	}
	// Mirror the C ABI only in the test; production writes the C record directly.
	type runtimeString struct{ Offset, Size uintptr }
	type runtimeSymbol struct {
		Name, SystemName, File runtimeString
		StartLine, Line        uint32
		Column                 int64
	}
	symbol := func(index int) *runtimeSymbol {
		offset := int(w.frame(0).SymbolsOffset) + index*int(unsafe.Sizeof(runtimeSymbol{}))
		return (*runtimeSymbol)(unsafe.Add(unsafe.Pointer(w.result.Arena), offset))
	}
	read := func(offset, size int) string {
		return string(unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(w.result.Arena), offset)), size))
	}
	got := symbol(0)
	if read(int(got.Name.Offset), int(got.Name.Size)) != "function" ||
		read(int(got.SystemName.Offset), int(got.SystemName.Size)) != systemName ||
		read(int(got.File.Offset), int(got.File.Size)) != "source" ||
		int32(got.StartLine) != 2 || int32(got.Line) != 8 || int64(got.Column) != 1<<40 {
		t.Fatalf("symbol metadata lost after arena growth: %+v", got)
	}
	plain := symbol(1)
	if read(int(plain.Name.Offset), int(plain.Name.Size)) != "plain" ||
		plain.SystemName.Size != 0 || plain.Column != 0 || plain.StartLine != 0 || plain.Line != 0 {
		t.Fatalf("plain call changed: %+v", plain)
	}
	if err := frame.AddInlineFunction("", "", 0, 0, "", 0); err == nil {
		t.Fatal("accepted more symbols than reserved")
	}
	result, err := w.Finish()
	if err != nil {
		t.Fatal(err)
	}
	w.result = result // Return ownership for Close.
}

func TestSystemNameRespectsStringBudget(t *testing.T) {
	w, err := New(1, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	frame, err := w.AddFrame(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	w.stringBytes = maxRuntimeStringBytes
	if err := frame.AddInlineFunction("", "", 0, 0, "overflow", 0); err == nil {
		t.Fatal("system name bypassed the arena budget")
	}
}
