//go:build cgo

package render

import (
	"bytes"
	"fmt"
	"io"

	pprof "github.com/google/pprof/profile"
	"google.golang.org/protobuf/proto"

	"github.com/yandex/perforator/perforator/pkg/cprofile"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/collapsed"
	"github.com/yandex/perforator/perforator/pkg/profile/flamegraph/convert"
	profilepb "github.com/yandex/perforator/perforator/proto/profile"
)

// CGOFlameGraph renders flamegraphs using the C++ trie-based implementation.
// It implements FlameGraphRenderer but returns errors for unsupported features.
type CGOFlameGraph struct {
	// Profile data (stored as pprof bytes)
	profileData []byte
	hasProfile  bool

	// Options
	format         Format
	title          string
	inverted       bool
	minWeight      float64
	maxDepth       int
	sampleType     string
	lineNumbers    bool
	fileNames      bool
	pathPrefix     string
	addressPolicy  AddressRenderPolicy
	ignoreFullPath bool
}

// Compile-time check that CGOFlameGraph implements FlameGraphRenderer.
var _ FlameGraphRenderer = (*CGOFlameGraph)(nil)

// NewCGOFlameGraph creates a new C++ trie-based flamegraph renderer.
func NewCGOFlameGraph() *CGOFlameGraph {
	return &CGOFlameGraph{
		format:     HTMLFormat,
		title:      "Flame Graph",
		sampleType: "", // Empty uses pprof default behavior: DefaultSampleType or last sample type
		fileNames:  true,
		pathPrefix: "@",
	}
}

func (f *CGOFlameGraph) AddProfile(p *pprof.Profile) error {
	if f.hasProfile {
		return fmt.Errorf("CGO renderer does not support adding multiple profiles")
	}

	// Resolve sample type and set DefaultSampleType so C++ renderer uses correct index
	sampleIndex, err := resolveSampleIndex(p, f.sampleType)
	if err != nil {
		return err
	}
	if sampleIndex < len(p.SampleType) {
		p.DefaultSampleType = p.SampleType[sampleIndex].Type
	}

	buf := new(bytes.Buffer)
	if err := p.WriteUncompressed(buf); err != nil {
		return fmt.Errorf("failed to serialize profile: %w", err)
	}
	f.profileData = buf.Bytes()
	f.hasProfile = true
	return nil
}

func (f *CGOFlameGraph) AddBaselineProfile(p *pprof.Profile) error {
	return fmt.Errorf("CGO renderer does not support baseline/diff profiles")
}

func (f *CGOFlameGraph) AddCollapsedProfile(p *collapsed.Profile) error {
	prof, err := convert.CollapsedToPProf(p)
	if err != nil {
		return fmt.Errorf("failed to convert collapsed to pprof: %w", err)
	}
	return f.AddProfile(prof)
}

func (f *CGOFlameGraph) AddCollapsedBaselineProfile(p *collapsed.Profile) error {
	return fmt.Errorf("CGO renderer does not support collapsed profiles")
}

func (f *CGOFlameGraph) SetFormat(format Format) {
	f.format = format
}

func (f *CGOFlameGraph) SetTitle(title string) {
	f.title = title
}

func (f *CGOFlameGraph) SetInverted(value bool) {
	f.inverted = value
}

func (f *CGOFlameGraph) SetMinWeight(value float64) {
	f.minWeight = value
}

func (f *CGOFlameGraph) SetDepthLimit(value int) {
	f.maxDepth = value
}

func (f *CGOFlameGraph) SetSampleType(typ string) {
	f.sampleType = typ
}

func (f *CGOFlameGraph) SetLineNumbers(value bool) {
	f.lineNumbers = value
}

func (f *CGOFlameGraph) SetFileNames(value bool) {
	f.fileNames = value
}

func (f *CGOFlameGraph) SetFilePathPrefix(value string) {
	f.pathPrefix = value
}

func (f *CGOFlameGraph) SetAddressRenderPolicy(policy AddressRenderPolicy) {
	f.addressPolicy = policy
}

func (f *CGOFlameGraph) SetIgnoreFullPath(value bool) {
	f.ignoreFullPath = value
}

func (f *CGOFlameGraph) Render(w io.Writer) error {
	// Validate unsupported options
	if f.inverted {
		return fmt.Errorf("CGO renderer does not support inverted flamegraphs")
	}
	if f.addressPolicy != "" && f.addressPolicy != RenderAddressesNever {
		return fmt.Errorf("CGO renderer does not support address rendering policy %s", f.addressPolicy)
	}
	if !f.hasProfile {
		return fmt.Errorf("no profile added")
	}
	if f.format == JSONPrettyFormat {
		return fmt.Errorf("CGO renderer does not support JSON pretty format")
	}

	// Build render options for C++
	opts := &profilepb.RenderOptions{}
	if f.maxDepth > 0 {
		opts.MaxDepth = proto.Uint32(uint32(f.maxDepth))
	}
	if f.minWeight > 0 {
		opts.MinWeight = proto.Float64(f.minWeight)
	}
	opts.ShowLineNumbers = proto.Bool(f.lineNumbers)
	opts.ShowFileNames = proto.Bool(f.fileNames)
	if f.pathPrefix != "" {
		opts.FilePathPrefix = proto.String(f.pathPrefix)
	}

	// Call C++ renderer
	jsonData, err := cprofile.RenderFlameGraphFromPProf(f.profileData, opts)
	if err != nil {
		return fmt.Errorf("C++ render failed: %w", err)
	}

	// Output based on format
	switch f.format {
	case JSONFormat:
		_, err = w.Write(jsonData)
		return err
	case HTMLFormat:
		return WrapJSONInHTML(jsonData, w)
	default:
		return fmt.Errorf("unsupported format: %s", f.format)
	}
}

func (f *CGOFlameGraph) TotalEvents() float64 {
	// Not available in CGO renderer without parsing the profile
	return 0
}
