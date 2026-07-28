package render

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"io"
	"strconv"
	"strings"

	pprof "github.com/google/pprof/profile"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/pkg/profile/labels"
)

//go:embed tmpl.html
var htmlTmpl string

var tmpl *template.Template

func init() {
	tmpl = template.Must(template.New("html").Parse(htmlTmpl))
}

type Format string

const (
	HTMLFormat       Format = "html"
	JSONFormat       Format = "json"
	JSONPrettyFormat Format = "json-pretty"

	PlainTextFormat Format = "text" // Used only in TextFormat struct
)

const (
	unsymbolizedFunction = "<unsymbolized function>"
	unknownMapping       = "<unknown mapping>"
	truncatedStack       = "(truncated stack)"
)

////////////////////////////////////////////////////////////////////////////////

type AddressRenderPolicy string

const (
	RenderAddressesNever        AddressRenderPolicy = "never"
	RenderAddressesUnsymbolized AddressRenderPolicy = "unsymbolized"
	RenderAddressesAlways       AddressRenderPolicy = "always"
)

var (
	AddressRenderPolicies = []AddressRenderPolicy{
		RenderAddressesNever,
		RenderAddressesUnsymbolized,
		RenderAddressesAlways,
	}
)

////////////////////////////////////////////////////////////////////////////////

type locationMeta struct {
	address   uint64
	mappingID uint64
}

type locationData struct {
	name    string
	file    string
	inlined bool
}

// LocationFrameOptions contains configuration for rendering location frames
type LocationFrameOptions struct {
	AddressPolicy  AddressRenderPolicy
	LineNumbers    bool
	FileNames      bool
	FilePathPrefix string
}

type FlameGraph struct {
	format   Format
	inverted bool

	locationFrameOptions LocationFrameOptions

	title      string
	maxDepth   int
	minWeight  float64
	frameType  string
	sampleType string // Sample type in type.unit format (e.g., cpu.cycles)

	locationsCache map[locationMeta][]locationData
	bb             *blocksBuilder
}

// Compile-time check that FlameGraph implements FlameGraphRenderer.
var _ FlameGraphRenderer = (*FlameGraph)(nil)

func NewFlameGraph() *FlameGraph {
	return &FlameGraph{
		locationFrameOptions: LocationFrameOptions{
			FileNames:      true,
			FilePathPrefix: "@",
		},
		format:         HTMLFormat,
		title:          "Flame Graph",
		frameType:      "Function",
		sampleType:     "",
		locationsCache: make(map[locationMeta][]locationData),
		bb:             newBlocksBuilder(),
	}
}

func (f *FlameGraph) SetInverted(value bool) {
	f.inverted = value
}

func (f *FlameGraph) SetIgnoreFullPath(value bool) {
	f.bb.SetIgnoreFullPath(value)
}

func (f *FlameGraph) SetTitle(value string) {
	f.title = value
}

func (f *FlameGraph) SetDepthLimit(value int) {
	f.maxDepth = value
}

func (f *FlameGraph) SetMinWeight(value float64) {
	f.minWeight = value
}

func (f *FlameGraph) SetFrameType(typ string) {
	f.frameType = typ
}

// SetSampleType sets the sample type in type.unit format (e.g., cpu.cycles).
// If empty, uses the first sample type from the profile.
func (f *FlameGraph) SetSampleType(typ string) {
	f.sampleType = typ
}

func (f *FlameGraph) SetLineNumbers(value bool) {
	f.locationFrameOptions.LineNumbers = value
}

func (f *FlameGraph) SetFileNames(value bool) {
	f.locationFrameOptions.FileNames = value
}

func (f *FlameGraph) SetFilePathPrefix(value string) {
	f.locationFrameOptions.FilePathPrefix = value
}

func (f *FlameGraph) SetFormat(format Format) {
	f.format = format
}

func (f *FlameGraph) SetAddressRenderPolicy(policy AddressRenderPolicy) {
	f.locationFrameOptions.AddressPolicy = policy
}

func (f *FlameGraph) AddProfile(profile *pprof.Profile) error {
	return f.addProfile(profile, false)
}

func (f *FlameGraph) AddBaselineProfile(profile *pprof.Profile) error {
	return f.addProfile(profile, true)
}

////////////////////////////////////////////////////////////////////////////////

// Render implements FlameGraphRenderer.
func (f *FlameGraph) Render(w io.Writer) error {
	blocks := f.bb.Finish(f.minWeight)
	return f.renderBlocks(blocks, w)
}

func (f *FlameGraph) RenderBytes() ([]byte, error) {
	var w bytes.Buffer
	err := f.Render(&w)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (f *FlameGraph) TotalEvents() float64 {
	return f.bb.root.nextCount.events
}

func (f *FlameGraph) RenderPProf(profile *pprof.Profile, w io.Writer) error {
	if err := f.AddProfile(profile); err != nil {
		return err
	}
	return f.Render(w)
}

func (f *FlameGraph) newBlocksJSONRenderer(blocks []*block) *BlocksJSONRenderer {
	return NewBlocksJSONRenderer(blocks, f.sampleType, f.frameType)
}

func (f *FlameGraph) renderBlocks(blocks []*block, w io.Writer) error {
	switch f.format {
	case JSONFormat:
		return f.newBlocksJSONRenderer(blocks).RenderJSON(w)
	case JSONPrettyFormat:
		return f.newBlocksJSONRenderer(blocks).RenderPrettyJSON(w)
	case HTMLFormat:
		return RenderJSONAsHTML(f.newBlocksJSONRenderer(blocks), w)
	default:
		return fmt.Errorf("unsupported format: %s", f.format)
	}
}

func getLocationFrames(loc *pprof.Location, options LocationFrameOptions) []locationData {
	frames := make([]locationData, 0, len(loc.Line))
	// loc.Line is innermost-first (pprof / profile.proto InlineChains
	// convention): the last entry is the physically-present, non-inlined
	// function; the rest are its inlined callees. Emit outermost-first so the
	// caller stays the flamegraph ancestor.
	for i := len(loc.Line) - 1; i >= 0; i-- {
		line := loc.Line[i]
		funcname := "??"
		if line.Function != nil {
			if line.Function.Name != "" {
				funcname = line.Function.Name
			} else {
				funcname = line.Function.SystemName
			}
		}

		if IsInvalidFunctionName(funcname) {
			funcname = unsymbolizedFunction
		}

		switch {
		case options.AddressPolicy == RenderAddressesUnsymbolized && funcname == unsymbolizedFunction:
			fallthrough
		case options.AddressPolicy == RenderAddressesAlways:
			funcname = fmt.Sprintf("{%#x} %s", loc.Address, funcname)
		}

		lineNumber := ""
		if options.LineNumbers && line.Line > 0 {
			lineNumber = fmt.Sprintf(":%d", line.Line)
		}

		filename := ""
		if line.Function != nil {
			filename = sanitizeFileName(line.Function.Filename)
		}
		if isInvalidFilename(filename) && loc.Mapping != nil {
			filename = loc.Mapping.File
		}
		if isInvalidFilename(filename) {
			filename = "??"
		}

		inlined := i != len(loc.Line)-1

		filepos := ""
		if options.FileNames {
			filepos = options.FilePathPrefix + filename + lineNumber
		}

		frames = append(frames, locationData{name: or(funcname), file: filepos, inlined: inlined})
	}

	return frames
}

func (f *FlameGraph) getLocationFramesCached(loc *pprof.Location) []locationData {
	if loc.Mapping == nil || loc.Mapping.BuildID == "" {
		return getLocationFrames(loc, f.locationFrameOptions)
	}

	meta := locationMeta{
		address:   loc.Address,
		mappingID: loc.Mapping.ID,
	}
	frames, found := f.locationsCache[meta]
	if !found {
		frames = getLocationFrames(loc, f.locationFrameOptions)
		f.locationsCache[meta] = frames
	}

	return frames
}

func (f *FlameGraph) clearLocationsCache() {
	f.locationsCache = make(map[locationMeta][]locationData)
}

// resolveSampleIndex resolves a sample type selector to an actual index.
// Selector can be a numeric index, "type", or "type.unit" format. Empty selector
// uses pprof default behavior: match DefaultSampleType by Type, or fall back to last sample type.
func resolveSampleIndex(p *pprof.Profile, selector string) (int, error) {
	if len(p.SampleType) == 0 {
		return 0, fmt.Errorf("profile has no sample types")
	}

	// Empty selector: use pprof default behavior
	if selector == "" {
		// First, try to find DefaultSampleType by Type field
		for i, st := range p.SampleType {
			if st.Type == p.DefaultSampleType {
				return i, nil
			}
		}
		// Fall back to last sample type (pprof behavior)
		return len(p.SampleType) - 1, nil
	}

	// Try numeric index
	if idx, err := strconv.Atoi(selector); err == nil {
		if idx < 0 || idx >= len(p.SampleType) {
			return 0, fmt.Errorf("sample type index %d out of range [0, %d)", idx, len(p.SampleType))
		}
		return idx, nil
	}

	// Try type.unit format
	if parts := strings.SplitN(selector, ".", 2); len(parts) == 2 {
		typeName, unitName := parts[0], parts[1]
		for i, st := range p.SampleType {
			if st.Type == typeName && st.Unit == unitName {
				return i, nil
			}
		}
		return 0, fmt.Errorf("sample type %q not found in profile", selector)
	}

	// Match by Type only (like pprof does)
	for i, st := range p.SampleType {
		if st.Type == selector {
			return i, nil
		}
	}
	return 0, fmt.Errorf("sample type %q not found in profile", selector)
}

func (f *FlameGraph) addProfile(p *pprof.Profile, baseline bool) error {
	defer func() {
		f.clearLocationsCache()
	}()

	sampleIndex, err := resolveSampleIndex(p, f.sampleType)
	if err != nil {
		return err
	}
	// Set sampleType to full type.unit format for display
	st := p.SampleType[sampleIndex]
	f.sampleType = st.Type + "." + st.Unit

	for _, sample := range p.Sample {
		procinfo := labels.ExtractProcessInfo(sample)

		iter := f.bb.MakeIterator(float64(sample.Value[sampleIndex]), baseline)
		for _, container := range procinfo.Containers {
			iter.Advance(container, "").SetKind("container")
		}
		if pid := procinfo.Pid; pid != nil {
			iter.Advance(fmt.Sprintf("%d", *pid), "").SetKind("process")
		}
		if name := procinfo.ProcessName; name != "" {
			iter.Advance(name, "").SetKind("process")
		}
		if name := procinfo.ThreadName; name != "" {
			iter.Advance(name, "").SetKind("thread")
		}
		for _, signal := range sample.Label["signal:name"] {
			iter.Advance(signal, "").SetKind("signal")
		}

		startdepth := iter.Depth()
		for i := len(sample.Location) - 1; i >= 0; i-- {
			loc := sample.Location[i]
			origin := FrameOriginNative
			if loc.Mapping != nil {
				switch loc.Mapping.File {
				case profile.KernelSpecialMapping:
					origin = FrameOriginKernel
				case profile.JVMSpecialMapping:
					origin = FrameOriginJVM
				case profile.PHPSpecialMapping:
					origin = FrameOriginPHP
				case profile.PythonSpecialMapping:
					origin = FrameOriginPython
				case profile.LuaSpecialMapping:
					origin = FrameOriginLua
				}
			}

			if len(loc.Line) == 0 {
				if f.maxDepth > 0 && iter.Depth() >= f.maxDepth {
					iter.Advance(truncatedStack, "").SetFrameOrigin(origin)
					goto done
				}

				if loc.Mapping == nil {
					// Skip lowest frames without mappings. They are useless.
					if iter.Depth() != startdepth {
						iter.Advance(unknownMapping, "").SetFrameOrigin(origin)
					}
				} else {
					name := "??"
					path := ""
					if f.locationFrameOptions.FileNames {
						path = loc.Mapping.File
					}
					iter.Advance(name, path).SetFrameOrigin(origin)
				}

				continue
			}

			frames := f.getLocationFramesCached(loc)
			for _, frame := range frames {
				if f.maxDepth > 0 && f.maxDepth < len(sample.Location) && iter.Depth() == f.maxDepth {
					iter.Advance(truncatedStack, "").SetFrameOrigin(origin)
					goto done
				}

				iter.
					Advance(frame.name, frame.file).
					SetInlined(frame.inlined).
					SetFrameOrigin(origin)
			}
		}
	done:
	}
	return nil
}

func sanitizeFileName(name string) string {
	if strings.HasPrefix(name, "/-B") || strings.HasPrefix(name, "/-S") {
		return name[3:]
	}

	return name
}

func or(x string) string {
	if x != "" {
		return x
	}
	return "??"
}

func IsInvalidFunctionName(funcname string) bool {
	return funcname == "" || funcname == "??" || funcname == "<invalid>" || funcname == "<undefined>"
}

func isInvalidFilename(filename string) bool {
	return filename == "" || filename == "??" || filename == "<invalid>" || filename == "<undefined>" || filename == "<unknown>"
}
