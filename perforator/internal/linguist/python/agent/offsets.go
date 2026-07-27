package agent

import (
	"embed"
	"encoding/json"
	"reflect"
	"regexp"

	"github.com/yandex/perforator/perforator/internal/linguist/common/offsetloader"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

const (
	UnspecifiedOffset = uint32((1 << 32) - 1)
)

//go:embed offsets/*.json
var offsetsFS embed.FS

var pythonOffsets *offsetloader.VersionOffsets[*unwinder.PythonInternalsOffsets]

// unfilledOffsets is a PythonInternalsOffsets with all numeric fields set to UnspecifiedOffset
var unfilledOffsets unwinder.PythonInternalsOffsets

// Structure to match the JSON format from extract_offsets.py
type jsonOffsets struct {
	PyThreadState      map[string]int `json:"PyThreadState"`
	PyInterpreterState map[string]int `json:"PyInterpreterState"`
	PyCodeObject       map[string]int `json:"PyCodeObject"`
	PyFrameObject      map[string]int `json:"PyFrameObject,omitempty"`
	PyRuntimeState     map[string]int `json:"_PyRuntimeState"`
	PyCFrame           map[string]int `json:"_PyCFrame,omitempty"`
	PyInterpreterFrame map[string]int `json:"_PyInterpreterFrame,omitempty"`
	PyASCIIObject      map[string]int `json:"PyASCIIObject,omitempty"`
	PyUnicodeObject    map[string]int `json:"PyUnicodeObject,omitempty"`
	PyStringObject     map[string]int `json:"PyStringObject,omitempty"`
	PyBytesObject      map[string]int `json:"PyBytesObject,omitempty"`
	PyTssT             map[string]int `json:"Py_tss_t,omitempty"`
}

func fillUnspecifiedOffsets(val reflect.Value) {
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		if field.Kind() == reflect.Uint32 {
			field.Set(reflect.ValueOf(UnspecifiedOffset))
		} else if field.Kind() == reflect.Uint64 {
			field.Set(reflect.ValueOf(UnspecifiedOffset))
		} else if field.Kind() == reflect.Struct {
			fillUnspecifiedOffsets(field)
		} else if field.Kind() == reflect.Uint8 {
			// Skip uint8 fields - they're for bit positions, not offsets
			continue
		}
	}
}

var pythonFilenamePattern = regexp.MustCompile(`cpython-(\d+\.\d+(?:\.\d+)?)-offsets\.json`)

func init() {
	fillUnspecifiedOffsets(reflect.ValueOf(&unfilledOffsets))
	pythonOffsets = offsetloader.Load(offsetsFS, "offsets", pythonFilenamePattern, parsePythonOffsets)
}

func parsePythonOffsets(data []byte) (*unwinder.PythonInternalsOffsets, error) {
	var j jsonOffsets
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	return convertToPythonInternalsOffsets(j), nil
}

// Extract PyThreadState offsets from JSON data
func extractPyThreadStateOffsets(data map[string]int) unwinder.PythonThreadStateOffsets {
	var offsets unwinder.PythonThreadStateOffsets

	if val, ok := data["next"]; ok {
		offsets.NextThread = uint32(val)
	} else {
		offsets.NextThread = UnspecifiedOffset
	}

	if val, ok := data["prev"]; ok {
		offsets.PrevThread = uint32(val)
	} else {
		offsets.PrevThread = UnspecifiedOffset
	}

	if val, ok := data["native_thread_id"]; ok {
		offsets.NativeThreadId = uint32(val)
	} else {
		offsets.NativeThreadId = UnspecifiedOffset
	}

	if val, ok := data["thread_id"]; ok {
		offsets.ThreadId = uint32(val)
	} else {
		offsets.ThreadId = UnspecifiedOffset
	}

	if val, ok := data["cframe"]; ok {
		offsets.Cframe = uint32(val)
	} else {
		offsets.Cframe = UnspecifiedOffset
	}

	if val, ok := data["current_frame"]; ok {
		offsets.CurrentFrame = uint32(val)
	} else if val, ok := data["frame"]; ok {
		// For CPython before 3.11
		offsets.CurrentFrame = uint32(val)
	} else {
		offsets.CurrentFrame = UnspecifiedOffset
	}

	return offsets
}

// Extract PyInterpreterState offsets from JSON data
func extractPyInterpreterStateOffsets(data map[string]int) unwinder.PythonInterpreterStateOffsets {
	var offsets unwinder.PythonInterpreterStateOffsets

	if val, ok := data["next"]; ok {
		offsets.Next = uint32(val)
	} else {
		offsets.Next = UnspecifiedOffset
	}

	if val, ok := data["threads.head"]; ok {
		offsets.ThreadsHead = uint32(val)
	} else {
		offsets.ThreadsHead = UnspecifiedOffset
	}

	return offsets
}

// Extract PyCodeObject offsets from JSON data
func extractPyCodeObjectOffsets(data map[string]int) unwinder.PythonCodeObjectOffsets {
	var offsets unwinder.PythonCodeObjectOffsets

	if val, ok := data["co_firstlineno"]; ok {
		offsets.CoFirstlineno = uint32(val)
	} else {
		offsets.CoFirstlineno = UnspecifiedOffset
	}

	if val, ok := data["co_filename"]; ok {
		offsets.Filename = uint32(val)
	} else {
		offsets.Filename = UnspecifiedOffset
	}

	if val, ok := data["co_qualname"]; ok {
		offsets.Name = uint32(val)
	} else if val, ok := data["co_name"]; ok {
		offsets.Name = uint32(val)
	} else {
		offsets.Name = UnspecifiedOffset
	}

	if val, ok := data["co_code_adaptive"]; ok {
		offsets.CoCodeAdaptive = uint32(val)
	} else {
		offsets.CoCodeAdaptive = UnspecifiedOffset
	}

	if val, ok := data["co_linetable"]; ok {
		offsets.CoLinetable = uint32(val)
	} else {
		offsets.CoLinetable = UnspecifiedOffset
	}

	return offsets
}

// Extract frame offsets from JSON data
func extractPyFrameOffsets(data map[string]int) unwinder.PythonFrameOffsets {
	var offsets unwinder.PythonFrameOffsets

	if val, ok := data["f_code"]; ok {
		offsets.FCode = uint32(val)
	} else if val, ok := data["f_executable"]; ok {
		// Python 3.13+ uses f_executable instead of f_code
		offsets.FCode = uint32(val)
	} else {
		offsets.FCode = UnspecifiedOffset
	}

	if val, ok := data["previous"]; ok {
		offsets.Previous = uint32(val)
	} else if val, ok := data["f_back"]; ok {
		offsets.Previous = uint32(val)
	} else {
		offsets.Previous = UnspecifiedOffset
	}

	if val, ok := data["owner"]; ok {
		offsets.Owner = uint32(val)
	} else {
		offsets.Owner = UnspecifiedOffset
	}

	// instr_ptr (CPython 3.11+): offset of the currently executing bytecode
	// instruction pointer within _PyInterpreterFrame. extract_offsets.py emits
	// this key for both `prev_instr` (3.11/3.12) and `instr_ptr` (3.13+).
	if val, ok := data["instr_ptr"]; ok {
		offsets.InstrPtr = uint32(val)
	} else {
		offsets.InstrPtr = UnspecifiedOffset
	}

	return offsets
}

// Extract PyBytesObject offsets from JSON data (CPython 3.11+).
func extractPyBytesObjectOffsets(data map[string]int) unwinder.PythonBytesObjectOffsets {
	var offsets unwinder.PythonBytesObjectOffsets

	if val, ok := data["ob_size"]; ok {
		offsets.ObSize = uint32(val)
	} else {
		offsets.ObSize = UnspecifiedOffset
	}

	if val, ok := data["ob_sval"]; ok {
		offsets.ObSval = uint32(val)
	} else {
		offsets.ObSval = UnspecifiedOffset
	}

	return offsets
}

// Extract PyCFrame offsets from JSON data
func extractPyCFrameOffsets(data map[string]int) unwinder.PythonCframeOffsets {
	var offsets unwinder.PythonCframeOffsets

	if val, ok := data["current_frame"]; ok {
		offsets.CurrentFrame = uint32(val)
	} else {
		offsets.CurrentFrame = UnspecifiedOffset
	}

	return offsets
}

// Extract PyRuntimeState offsets from JSON data
func extractPyRuntimeStateOffsets(data map[string]int) unwinder.PythonRuntimeStateOffsets {
	var offsets unwinder.PythonRuntimeStateOffsets

	if val, ok := data["interpreters.main"]; ok {
		offsets.PyInterpretersMain = uint32(val)
	} else {
		offsets.PyInterpretersMain = UnspecifiedOffset
	}

	return offsets
}

// Extract PyUnicodeObject offsets from JSON data
func extractPyUnicodeObjectOffsets(data map[string]int) unwinder.PythonStringObjectOffsets {
	var offsets unwinder.PythonStringObjectOffsets

	if val, ok := data["length"]; ok {
		offsets.Length = uint32(val)
	} else {
		offsets.Length = UnspecifiedOffset
	}

	if val, ok := data["str"]; ok {
		offsets.Data = uint32(val)
	} else {
		offsets.Data = UnspecifiedOffset
	}

	return offsets
}

// Extract PyStringObject offsets from JSON data
func extractPyStringObjectOffsets(data map[string]int) unwinder.PythonStringObjectOffsets {
	var offsets unwinder.PythonStringObjectOffsets

	if val, ok := data["ob_size"]; ok {
		offsets.Length = uint32(val)
	} else {
		offsets.Length = UnspecifiedOffset
	}

	if val, ok := data["ob_sval"]; ok {
		offsets.Data = uint32(val)
	} else {
		offsets.Data = UnspecifiedOffset
	}

	return offsets
}

// Extract PyASCIIObject offsets from JSON data
func extractPyASCIIObjectOffsets(data map[string]int) unwinder.PythonStringObjectOffsets {
	var offsets unwinder.PythonStringObjectOffsets

	if val, ok := data["length"]; ok {
		offsets.Length = uint32(val)
	} else {
		offsets.Length = UnspecifiedOffset
	}

	if val, ok := data["state"]; ok {
		offsets.State = uint32(val)
	} else {
		offsets.State = UnspecifiedOffset
	}

	if val, ok := data["data"]; ok {
		offsets.Data = uint32(val)
	} else {
		offsets.Data = UnspecifiedOffset
	}

	// Use the bit flags from the JSON if available, otherwise use defaults
	if val, ok := data["ascii_bit"]; ok {
		offsets.AsciiBit = uint8(val)
	} else {
		offsets.AsciiBit = 6 // Default
	}

	if val, ok := data["compact_bit"]; ok {
		offsets.CompactBit = uint8(val)
	} else {
		offsets.CompactBit = 5 // Default
	}

	if val, ok := data["static_bit"]; ok {
		offsets.StaticallyAllocatedBit = uint8(val)
	} else {
		offsets.StaticallyAllocatedBit = 7 // Default
	}

	return offsets
}

// Extract Py_tss_t offsets from JSON data
func extractPyTssTOffsets(data map[string]int) unwinder.PythonTssTOffsets {
	var offsets unwinder.PythonTssTOffsets

	if val, ok := data["_is_initialized"]; ok {
		offsets.IsInitialized = uint32(val)
	} else {
		offsets.IsInitialized = UnspecifiedOffset
	}

	if val, ok := data["_key"]; ok {
		offsets.Key = uint32(val)
	} else {
		offsets.Key = UnspecifiedOffset
	}

	return offsets
}

// Convert JSON offsets to PythonInternalsOffsets
func convertToPythonInternalsOffsets(data jsonOffsets) *unwinder.PythonInternalsOffsets {
	offsets := &unwinder.PythonInternalsOffsets{}
	*offsets = unfilledOffsets

	// Extract offsets for each Python structure
	if data.PyThreadState != nil {
		offsets.PyThreadStateOffsets = extractPyThreadStateOffsets(data.PyThreadState)
	}

	if data.PyInterpreterState != nil {
		offsets.PyInterpreterStateOffsets = extractPyInterpreterStateOffsets(data.PyInterpreterState)
	}

	if data.PyCodeObject != nil {
		offsets.PyCodeObjectOffsets = extractPyCodeObjectOffsets(data.PyCodeObject)
	}

	if data.PyInterpreterFrame != nil {
		offsets.PyFrameOffsets = extractPyFrameOffsets(data.PyInterpreterFrame)
	} else if data.PyFrameObject != nil {
		offsets.PyFrameOffsets = extractPyFrameOffsets(data.PyFrameObject)
	}

	if data.PyCFrame != nil {
		offsets.PyCframeOffsets = extractPyCFrameOffsets(data.PyCFrame)
	}

	if data.PyRuntimeState != nil {
		offsets.PyRuntimeStateOffsets = extractPyRuntimeStateOffsets(data.PyRuntimeState)
	}

	if data.PyASCIIObject != nil {
		offsets.PyStringObjectOffsets = extractPyASCIIObjectOffsets(data.PyASCIIObject)
	} else if data.PyStringObject != nil {
		offsets.PyStringObjectOffsets = extractPyStringObjectOffsets(data.PyStringObject)
	} else if data.PyUnicodeObject != nil {
		offsets.PyStringObjectOffsets = extractPyUnicodeObjectOffsets(data.PyUnicodeObject)
	}

	if data.PyTssT != nil {
		offsets.PyTssTOffsets = extractPyTssTOffsets(data.PyTssT)
	}

	if data.PyBytesObject != nil {
		offsets.PyBytesObjectOffsets = extractPyBytesObjectOffsets(data.PyBytesObject)
	}

	return offsets
}
