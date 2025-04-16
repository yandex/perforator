package agent

import (
	"embed"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/python"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

const (
	UnspecifiedOffset = uint32((1 << 32) - 1)
)

//go:embed offsets/*.json
var offsetsFS embed.FS

// Map from Python version (encoded as uint32) to offsets
var pythonVersionOffsets map[uint32]*unwinder.PythonInternalsOffsets

// Structure to match the JSON format from extract_offsets.py
type jsonOffsets struct {
	PyThreadState      map[string]int `json:"PyThreadState"`
	PyInterpreterState map[string]int `json:"PyInterpreterState"`
	PyCodeObject       map[string]int `json:"PyCodeObject"`
	PyFrameObject      map[string]int `json:"PyFrameObject,omitempty"`
	PyRuntimeState     map[string]int `json:"_PyRuntimeState"`
	PyCFrame           map[string]int `json:"_PyCFrame,omitempty"`
	PyInterpreterFrame map[string]int `json:"_PyInterpreterFrame,omitempty"`
	PyASCIIObject      map[string]int `json:"PyASCIIObject"`
	PyTssT             map[string]int `json:"Py_tss_t,omitempty"`
}

// Convert a version string (major.minor.micro) to an encoded uint32
func encodeVersionFromString(version string) uint32 {
	parts := strings.Split(version, ".")
	if len(parts) < 3 {
		return 0
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}

	micro, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0
	}

	// Create a PythonVersion struct and use encodeVersion
	pythonVersion := &python.PythonVersion{
		Major: uint32(major),
		Minor: uint32(minor),
		Micro: uint32(micro),
	}

	return encodeVersion(pythonVersion)
}

// Init function to load all JSON files and build the offsets map
func init() {
	pythonVersionOffsets = make(map[uint32]*unwinder.PythonInternalsOffsets)

	// Read all files from the embedded filesystem
	entries, err := offsetsFS.ReadDir("offsets")
	if err != nil {
		panic(fmt.Sprintf("Failed to read offsets directory: %v", err))
	}

	// Compile the regex pattern once
	versionPattern := regexp.MustCompile(`cpython-(\d+\.\d+\.\d+)-offsets\.json`)

	// Parse each file
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			// Parse the version from the filename
			matches := versionPattern.FindStringSubmatch(entry.Name())
			if len(matches) < 2 {
				continue // Skip files that don't match the pattern
			}

			versionStr := matches[1]
			versionParts := strings.Split(versionStr, ".")
			if len(versionParts) < 3 {
				continue // Skip invalid versions
			}

			// Read the file content
			jsonData, err := offsetsFS.ReadFile(path.Join("offsets", entry.Name()))
			if err != nil {
				panic(fmt.Sprintf("Failed to read offset file %s: %v", entry.Name(), err))
			}

			// Parse the JSON into offsets
			var data jsonOffsets
			if err := json.Unmarshal(jsonData, &data); err != nil {
				panic(fmt.Sprintf("Failed to parse JSON from %s: %v", entry.Name(), err))
			}

			// Convert to PythonInternalsOffsets
			offsets := convertToPythonInternalsOffsets(data)

			// Store by encoded version
			versionKey := encodeVersionFromString(versionStr)
			pythonVersionOffsets[versionKey] = offsets
		}
	}
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

	return offsets
}

// Extract frame offsets from JSON data
func extractPyFrameOffsets(data map[string]int) unwinder.PyFrameOffsets {
	var offsets unwinder.PyFrameOffsets

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

// Extract PyASCIIObject offsets from JSON data
func extractPyASCIIObjectOffsets(data map[string]int) unwinder.PythonAsciiObjectOffsets {
	var offsets unwinder.PythonAsciiObjectOffsets

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
		offsets.PyAsciiObjectOffsets = extractPyASCIIObjectOffsets(data.PyASCIIObject)
	}

	if data.PyTssT != nil {
		offsets.PyTssTOffsets = extractPyTssTOffsets(data.PyTssT)
	}

	return offsets
}
