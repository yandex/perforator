package agent

import (
	"embed"
	"encoding/json"
	"errors"
	"regexp"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/php"
	"github.com/yandex/perforator/perforator/internal/linguist/common/offsetloader"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

//go:embed offsets/*.json
var offsetsFS embed.FS

var phpOffsets *offsetloader.VersionOffsets[*unwinder.PhpInternalsOffsets]

// JSON schema matching the output of extract_offsets.py.
type jsonOffsets struct {
	ZendExecutorGlobals map[string]int `json:"zend_executor_globals"`
	ZendExecuteData     map[string]int `json:"zend_execute_data"`
	ZendFunction        map[string]int `json:"zend_function"`
	ZendOpArray         map[string]int `json:"zend_op_array"`
	ZendString          map[string]int `json:"zend_string"`
}

var phpFilenamePattern = regexp.MustCompile(`php-(\d+\.\d+(?:\.\d+)?)-offsets\.json`)

func init() {
	phpOffsets = offsetloader.Load(offsetsFS, "offsets", phpFilenamePattern, parsePhpOffsets)
}

func parsePhpOffsets(data []byte) (*unwinder.PhpInternalsOffsets, error) {
	var j jsonOffsets
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}

	offsets := &unwinder.PhpInternalsOffsets{}

	if v, ok := j.ZendExecutorGlobals["current_execute_data"]; ok {
		offsets.ZendExecuteData = uint32(v)
	}

	if v, ok := j.ZendExecuteData["func"]; ok {
		offsets.ExecuteData.Function = uint32(v)
	}
	if v, ok := j.ZendExecuteData["This.u1.type_info"]; ok {
		offsets.ExecuteData.ThisTypeInfo = uint32(v)
	}
	if v, ok := j.ZendExecuteData["prev_execute_data"]; ok {
		offsets.ExecuteData.PrevExecuteData = uint32(v)
	}

	if v, ok := j.ZendFunction["type"]; ok {
		offsets.Function.Type = uint32(v)
	}
	if v, ok := j.ZendFunction["common.function_name"]; ok {
		offsets.Function.CommonFuncname = uint32(v)
	}

	if v, ok := j.ZendOpArray["filename"]; ok {
		offsets.Function.OpArray.Filename = uint32(v)
	}
	if v, ok := j.ZendOpArray["line_start"]; ok {
		offsets.Function.OpArray.Linestart = uint32(v)
	}

	if v, ok := j.ZendString["len"]; ok {
		offsets.ZendString.Len = uint32(v)
	}
	if v, ok := j.ZendString["val"]; ok {
		offsets.ZendString.Val = uint32(v)
	}

	return offsets, nil
}

// GetOffsets returns PHP struct offsets for the given version.
// Falls back to the nearest patch release within the same major.minor.
func GetOffsets(version *php.PhpVersion) (*unwinder.PhpInternalsOffsets, error) {
	if version == nil {
		return nil, errors.New("nil PHP version")
	}
	return phpOffsets.Get(offsetloader.NewLanguageVersion(version.Major, version.Minor, version.Release))
}
