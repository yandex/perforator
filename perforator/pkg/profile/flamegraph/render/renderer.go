package render

import (
	"io"

	pprof "github.com/google/pprof/profile"
)

// FlameGraphRenderer renders flamegraphs from profiles.
// This interface is implemented by both the Go blocks-based renderer (FlameGraph)
// and the C++ trie-based renderer (CGOFlameGraph).
//
// Input is always pprof. Other representations (e.g. collapsed/folded) are converted
// to pprof by the `convert` package before rendering, so the renderer has a single
// input shape.
type FlameGraphRenderer interface {
	// Profile input
	AddProfile(p *pprof.Profile) error
	AddBaselineProfile(p *pprof.Profile) error

	// Display options
	SetFormat(format Format)
	SetTitle(title string)
	SetInverted(value bool)
	SetMinWeight(value float64)
	SetDepthLimit(value int)
	SetSampleType(typ string)

	// Location display options
	SetLineNumbers(value bool)
	SetFileNames(value bool)
	SetFilePathPrefix(value string)
	SetAddressRenderPolicy(policy AddressRenderPolicy)
	SetIgnoreFullPath(value bool)

	// Output
	Render(w io.Writer) error
	TotalEvents() float64
}
