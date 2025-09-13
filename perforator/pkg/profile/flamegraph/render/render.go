package render

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"image/color"
	"io"
	"math"
	"slices"
	"sort"
	"strings"

	pprof "github.com/google/pprof/profile"
	"golang.org/x/exp/maps"

	"github.com/yandex/perforator/library/go/core/resource"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/render/format"
	"github.com/yandex/perforator/perforator/pkg/profile/labels"
)

//go:embed tmpl.html
var htmlTmpl string

//go:embed new_templ.html
var newHtmlTmpl string

var tmpl *template.Template

func init() {
	tmpl = template.New("root").Funcs(template.FuncMap{
		"add": func(a float64, b ...float64) float64 {
			for _, x := range b {
				a += x
			}
			return a
		},
		"sub": func(a float64, b ...float64) float64 {
			for _, x := range b {
				a -= x
			}
			return a
		},
		"mul": func(a, b float64) float64 {
			return a * b
		},
		"div": func(a, b float64) float64 {
			return a / b
		},
	})

	template.Must(tmpl.New("html").Parse(htmlTmpl))
	template.Must(tmpl.New("html-v2").Parse(newHtmlTmpl))
}

type Format string

const (
	HTMLFormat       Format = "html"
	HTMLFormatV2     Format = "html-v2"
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
	diff     bool

	locationFrameOptions LocationFrameOptions

	title     string
	maxDepth  int
	minWeight float64
	frameType string
	eventType string

	width               float64
	blockHeight         float64
	blockVerticalMargin float64

	fontSize  float64
	fontWidth float64

	padX float64

	locationsCache map[locationMeta][]locationData
	bb             *blocksBuilder
	blocks         []*block
	diffmult       float64
}

func NewFlameGraph() *FlameGraph {
	return &FlameGraph{
		locationFrameOptions: LocationFrameOptions{
			FileNames:      true,
			FilePathPrefix: "@",
		},
		format:              HTMLFormatV2,
		title:               "Flame Graph",
		frameType:           "Function",
		eventType:           "cycles",
		width:               1200,
		blockHeight:         15.0,
		blockVerticalMargin: 1.0,

		fontSize:  12.0,
		fontWidth: 0.59,

		padX:           10.0,
		locationsCache: make(map[locationMeta][]locationData),
		bb:             newBlocksBuilder(),
	}
}

func (f *FlameGraph) SetInverted(value bool) {
	f.inverted = value
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

func (f *FlameGraph) SetSampleType(typ string) {
	f.eventType = typ
}

func (f *FlameGraph) SetWidth(value float64) {
	f.width = value
}

func (f *FlameGraph) SetFontSize(size float64) {
	f.fontSize = size
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

func reverse(s string) string {
	runes := []rune(s)
	slices.Reverse(runes)
	return string(runes)
}

func (f *FlameGraph) namehash(name string) float64 {
	vector := 0.0
	weight := 1.0
	max := 1.0
	mod := 10
	for _, c := range name {
		i := int(c) % mod

		vector += float64(i) / float64(mod-1) * weight
		mod += 1
		max += 1 * weight
		weight *= 0.7

		if mod > 13 {
			break
		}
	}
	return (1.0 - vector/max)
}

func (f *FlameGraph) hashcolor(name string, module FrameOrigin) color.RGBA {
	v1 := f.namehash(name)
	v2 := f.namehash(reverse(name))
	v3 := v2

	switch module {
	case FrameOriginKernel:
		return color.RGBA{
			R: uint8(96 + 55*v2),
			G: uint8(96 + (255-96)*v1),
			B: uint8(205 + 50*v3),
			A: 0,
		}
	case FrameOriginPython:
		return color.RGBA{
			R: uint8(103 + 50*v2),
			G: uint8(178 + 77*v1),
			B: uint8(120 + 50*v3),
			A: 0,
		}
	default:
		return color.RGBA{
			R: uint8(205 + 50*v3),
			G: uint8(0 + 230*v1),
			B: uint8(0 + 55*v2),
			A: 0,
		}
	}
}

// Copy-pase from https://github.com/yandex/perforator/arcadia/yabs/poormansprofiler/flames/lib/__init__.py?blame=true&rev=r14194743#L170-185
func (f *FlameGraph) diffcolor(node *block) color.RGBA {
	lhs, rhs := node.nextCount.events, node.prevCount.events*f.diffmult
	diff := (lhs - rhs) / rhs
	d := min(math.Abs(diff), 1.)

	if d < 0.001 {
		hash := f.namehash(node.name)
		value := 180 + uint8(hash*60)
		return color.RGBA{value, value, value, 0}
	}

	var hoff, hpow, hcoef = 0.16, 4.0, -0.14
	if diff <= 0 {
		hoff, hpow, hcoef = 0.58, 2.0, 0.10
	}
	var soff, spow, scoef = 0.0, 4.5, 0.75

	h := hoff + math.Pow(d, 1./hpow)*hcoef
	s := soff + math.Pow(d, 1./spow)*scoef
	v := 1.0

	rgb := HSV(h*360, s, v)
	return rgb
}

func (f *FlameGraph) color(block *block) color.RGBA {
	if f.diff {
		return f.diffcolor(block)
	}

	return f.hashcolor(block.name, block.frameOrigin)
}

////////////////////////////////////////////////////////////////////////////////

func (f *FlameGraph) AddProfile(profile *pprof.Profile) error {
	f.addProfile(profile, false)
	return nil
}

func (f *FlameGraph) AddBaselineProfile(profile *pprof.Profile) error {
	f.diff = true
	f.addProfile(profile, true)
	return nil
}

func (f *FlameGraph) AddCollapsedProfile(profile *collapsed.Profile) error {
	f.addCollapsedProfile(profile, false)
	return nil
}

func (f *FlameGraph) AddCollapsedBaselineProfile(profile *collapsed.Profile) error {
	f.diff = true
	f.addCollapsedProfile(profile, true)
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func (f *FlameGraph) Render(w io.Writer) error {
	f.blocks = f.bb.Finish(f.minWeight)
	return f.renderBlocks(f.blocks, w)
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

func (f *FlameGraph) RenderCollapsed(profile *collapsed.Profile, w io.Writer) error {
	if err := f.AddCollapsedProfile(profile); err != nil {
		return err
	}
	return f.Render(w)
}

////////////////////////////////////////////////////////////////////////////////

type frame struct {
	FullText        int
	RectX           float64
	RectWidth       float64
	Level           int
	Color           color.RGBA
	FillStyle       int
	EventCount      float64
	SampleCount     int64
	BaseEventCount  float64
	BaseSampleCount int64
}

// Yes, this is ugly, but we have a LOT of frames, and rendering them through template engine is really slow,
// dozens of seconds slow.
func renderFramesByHand(frameLevels [][]*frame, diff bool) string {
	w := strings.Builder{}

	renderField := func(selector func(*frame) any, frames []*frame) {
		fmt.Fprint(&w, "[")
		for _, frame := range frames {
			fmt.Fprint(&w, selector(frame))
			fmt.Fprint(&w, ",")
		}
		fmt.Fprint(&w, "],\n")
	}

	fmt.Fprint(&w, "[\n")
	for _, frameLevel := range frameLevels {
		fmt.Fprint(&w, "[\n")

		renderField(func(f *frame) any { return f.RectX }, frameLevel)
		renderField(func(f *frame) any { return f.RectWidth }, frameLevel)
		renderField(func(f *frame) any { return f.FullText }, frameLevel)
		renderField(func(f *frame) any { return f.EventCount }, frameLevel)
		renderField(func(f *frame) any { return f.SampleCount }, frameLevel)
		renderField(func(f *frame) any { return f.FillStyle }, frameLevel)

		if diff {
			renderField(func(f *frame) any { return f.BaseEventCount }, frameLevel)
			renderField(func(f *frame) any { return f.BaseSampleCount }, frameLevel)
		}

		fmt.Fprint(&w, "],\n")
	}
	fmt.Fprint(&w, "]")

	return w.String()
}

func (f *FlameGraph) renderBlocksToPrettyJSON(blocks []*block, w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return f.encodeBlocksToJSON(blocks, enc)
}
func (f *FlameGraph) renderBlocksToJSON(blocks []*block, w io.Writer) error {
	enc := json.NewEncoder(w)
	return f.encodeBlocksToJSON(blocks, enc)
}

func populateWithIndexes(root *block, depth, queueSize int) [][]*block {
	// every block will be in the queue once;
	// the queue becomes smaller by cutting off its first element
	// the underlying array elements will stay in place and we will move over that section of memory
	// so we cannot allocate less than the amount of blocks
	q := make([]*block, 0, queueSize)
	q = append(q, root)
	blocksByLevels := make([][]*block, depth)
	lastLevel := 0
	lastIndex := 0
	for len(q) != 0 {
		currentBlock := q[0]
		q = q[1:]
		if currentBlock.level > lastLevel {
			lastLevel = currentBlock.level
			lastIndex = 0
		}
		currentBlock.setLevelPos(lastIndex)
		lastIndex += 1
		blocksByLevels[lastLevel] = append(blocksByLevels[lastLevel], currentBlock)
		children := currentBlock.children
		keys := maps.Keys(children)
		slices.Sort(keys)
		for _, key := range keys {
			q = append(q, children[key])
		}
	}
	return blocksByLevels
}

func (f *FlameGraph) encodeBlocksToJSON(blocks []*block, enc *json.Encoder) error {
	strtab := NewStringTable()

	maxLevel := 0
	for _, block := range blocks {
		if block.level > maxLevel {
			maxLevel = block.level
		}
	}

	nodeLevels := make([][]format.RenderingNode, maxLevel+1)

	blocksByLevels := populateWithIndexes(blocks[0], maxLevel+1, len(blocks))

	for _, blocksOnLevel := range blocksByLevels {
		for _, currentBlock := range blocksOnLevel {
			parentIndex := -1
			if currentBlock.parent != nil {
				parentIndex = currentBlock.parent.levelPos
			}
			node := format.RenderingNode{
				ParentIndex:     parentIndex,
				TextID:          strtab.Add(currentBlock.name),
				SampleCount:     currentBlock.nextCount.count,
				EventCount:      currentBlock.nextCount.events,
				BaseEventCount:  currentBlock.prevCount.events,
				BaseSampleCount: currentBlock.prevCount.count,
				FrameOrigin:     strtab.Add(string(currentBlock.frameOrigin)),
				Kind:            strtab.Add(currentBlock.kind),
				File:            strtab.Add(currentBlock.file),
				Inlined:         currentBlock.inlined,
			}
			nodeLevels[currentBlock.level] = append(nodeLevels[currentBlock.level], node)
		}
	}

	profileMeta := format.ProfileMeta{
		EventType: strtab.Add(f.eventType),
		FrameType: strtab.Add(f.frameType),
		Version:   2,
	}

	profileData := format.ProfileData{
		Nodes:   nodeLevels,
		Strings: strtab.Table(),
		Meta:    profileMeta,
	}

	err := enc.Encode(profileData)
	if err != nil {
		return err
	}

	return nil
}

func (f *FlameGraph) renderBlocksToHTMLV2(blocks []*block, w io.Writer) error {
	jsonBytes := make([]byte, 0, 1024)
	buf := bytes.NewBuffer(jsonBytes)
	err := f.renderBlocksToJSON(blocks, buf)
	if err != nil {
		return err
	}
	jsCode := template.HTML("<script>" + string(resource.Get("viewer.js")) + "</script>")

	jsonData := template.HTML("<script>window.__data__=" + buf.String() + "</script>")

	return tmpl.ExecuteTemplate(w, string(f.format), &struct {
		Json   template.HTML
		Script template.HTML
	}{
		Json:   jsonData,
		Script: jsCode,
	})

}

func (f *FlameGraph) renderBlocks(blocks []*block, w io.Writer) error {
	switch f.format {
	case JSONFormat:
		return f.renderBlocksToJSON(blocks, w)
	case JSONPrettyFormat:
		return f.renderBlocksToPrettyJSON(blocks, w)
	case HTMLFormat:
		return f.renderBlocksToHTML(blocks, w)
	case HTMLFormatV2:
		return f.renderBlocksToHTMLV2(blocks, w)
	default:
		return fmt.Errorf("unsupported format: %s", f.format)
	}
}

func (f *FlameGraph) renderBlocksToHTML(blocks []*block, w io.Writer) error {
	strtab := NewStringTable()

	maxLevel := 0
	for _, block := range blocks {
		if block.level > maxLevel {
			maxLevel = block.level
		}
	}

	padTop := f.fontSize * 3

	canvasWidth := f.width - 2.0*f.padX
	canvasHeight := (f.blockHeight + f.blockVerticalMargin) * float64(1+maxLevel)

	frames := make([]frame, 0, len(blocks))
	total := blocks[0].nextCount.events
	if blocks[0].prevCount.events >= 0 {
		f.diffmult = total / blocks[0].prevCount.events
	} else {
		f.diffmult = 1.0
	}

	for _, block := range blocks {
		// Skip disappeared (present in baseline, but not in the diff profile) blocks
		if block.weight == 0.0 {
			continue
		}

		x := f.padX + block.offset*canvasWidth
		y := padTop
		if f.inverted {
			y += float64(block.level) * (f.blockHeight + f.blockVerticalMargin)
		} else {
			y += canvasHeight - float64(1+block.level)*(f.blockHeight+f.blockVerticalMargin)
		}

		w := block.weight * canvasWidth

		color := f.color(block)
		fillStyle := fmt.Sprintf("#%02x%02x%02x", color.R, color.G, color.B)

		fullname := blockToString(block)

		res := frame{
			FullText:        strtab.Add(fullname),
			RectX:           x,
			RectWidth:       w,
			Level:           block.level,
			Color:           color,
			FillStyle:       strtab.Add(fillStyle),
			EventCount:      block.nextCount.events,
			SampleCount:     block.nextCount.count,
			BaseEventCount:  block.prevCount.events,
			BaseSampleCount: block.prevCount.count,
		}
		frames = append(frames, res)
	}

	sort.Slice(frames, func(i, j int) bool {
		if frames[i].RectX == frames[j].RectX {
			return frames[i].Level < frames[j].Level
		}
		return frames[i].RectX < frames[j].RectX
	})

	frameLevels := make([][]*frame, maxLevel+1)
	for i, frame := range frames {
		frameLevels[frame.Level] = append(frameLevels[frame.Level], &frames[i])
	}

	return tmpl.ExecuteTemplate(w, string(f.format), &struct {
		Diff                    bool
		Inverted                bool
		Title                   string
		EventType               string
		Frames                  []frame
		FrameLevels             [][]*frame
		Strings                 []string
		HandRenderedFrameLevels template.JS
	}{
		Diff:      f.diff,
		Inverted:  f.inverted,
		Title:     f.title,
		EventType: f.eventType,
		// NOTE: is only used for {{len .Frames}} in tmpl.html
		// but was actively used in SVG
		// probably should be removed from args later
		Frames:                  frames,
		FrameLevels:             frameLevels,
		Strings:                 strtab.Table(),
		HandRenderedFrameLevels: template.JS(renderFramesByHand(frameLevels, f.diff)),
	})
}

////////////////////////////////////////////////////////////////////////////////

// Best-effort attempts to guess origins of collapsed frames.
func guessCollapsedFrameOrigin(name string) FrameOrigin {
	if strings.HasSuffix(name, "[kernel]") {
		return FrameOriginKernel
	}
	if strings.HasSuffix(name, ".py") {
		return FrameOriginPython
	}
	return FrameOriginNative
}

func (f *FlameGraph) addCollapsedProfile(profile *collapsed.Profile, baseline bool) {
	for _, sample := range profile.Samples {
		iter := f.bb.MakeIterator(float64(sample.Value), baseline)
		for i, name := range sample.Stack {
			origin := guessCollapsedFrameOrigin(name)

			if f.maxDepth > 0 && f.maxDepth < len(sample.Stack) && i+1 == f.maxDepth {
				iter.Advance(truncatedStack, "").SetFrameOrigin(origin)
				break
			}
			iter.Advance(name, "").SetFrameOrigin(origin)
		}
	}
}

func getLocationFrames(loc *pprof.Location, options LocationFrameOptions) []locationData {
	frames := make([]locationData, 0, len(loc.Line))
	for i, line := range loc.Line {
		funcname := "??"
		if line.Function != nil {
			if line.Function.Name != "" {
				funcname = line.Function.Name
			} else {
				funcname = line.Function.SystemName
			}
		}

		if isInvalidFunctionName(funcname) {
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

		inlined := i > 0

		filepos := ""
		if options.FileNames {
			filepos = options.FilePathPrefix + filename + lineNumber
		}

		frames = append(frames, locationData{name: or(funcname), file: filepos, inlined: inlined})
	}

	return frames
}

func (f *FlameGraph) getLocationFramesCached(loc *pprof.Location) []locationData {
	if loc.Mapping == nil {
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

func (f *FlameGraph) addProfile(p *pprof.Profile, baseline bool) {
	defer func() {
		f.clearLocationsCache()
	}()

	sampleIndex := 0
	for i, name := range p.SampleType {
		if name.Type == p.DefaultSampleType {
			sampleIndex = i
		}
	}
	f.SetSampleType(p.SampleType[sampleIndex].Unit)

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
		locs := sample.Location
		slices.Reverse(locs)
		for _, loc := range locs {
			var origin FrameOrigin
			if loc.Mapping != nil && strings.Contains(loc.Mapping.File, "kernel") {
				origin = FrameOriginKernel
			}
			if loc.Mapping != nil && loc.Mapping.File == profile.PythonSpecialMapping {
				origin = FrameOriginPython
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

func isInvalidFunctionName(funcname string) bool {
	return funcname == "" || funcname == "??" || funcname == "<invalid>" || funcname == "<undefined>"
}

func isInvalidFilename(filename string) bool {
	return filename == "" || filename == "??" || filename == "<invalid>" || filename == "<undefined>" || filename == "<unknown>"
}

func blockToString(b *block) string {
	fullname := b.name

	if b.file != "" {
		fullname += fmt.Sprintf(" %s", b.file)
	} else if b.frameOrigin != "" && b.frameOrigin != FrameOriginNative {
		fullname += fmt.Sprintf(" [%s]", b.frameOrigin)
	} else if b.kind != "" {
		fullname += fmt.Sprintf(" (%s)", b.kind)
	}

	if b.inlined {
		fullname += " (inlined)"
	}

	return fullname
}
