package mvt

import (
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/clip"
)

var (
	// MapboxGLDefaultExtentBound holds the default mapbox vector tile bounds used by mapbox-gl.
	// (https://www.mapbox.com/mapbox-gl-js/style-spec/#sources-vector)
	MapboxGLDefaultExtentBound = orb.Bound{
		Min: orb.Point{-1 * DefaultExtent, -1 * DefaultExtent},
		Max: orb.Point{2*DefaultExtent - 1, 2*DefaultExtent - 1},
	}
)

// Clip will clip all geometries in all layers to the given bounds.
func (ls Layers) Clip(box orb.Bound) {
	for _, l := range ls {
		l.Clip(box)
	}
}

// Clip will clip all geometries in this layer to the given bounds.
// Will remove features that clip to an empty geometry, modifies the
// layer.Features slice in place.
func (l *Layer) Clip(box orb.Bound) {
	at := 0
	for _, f := range l.Features {
		g := clip.Geometry(box, f.Geometry)
		if g != nil {
			f.Geometry = g
			l.Features[at] = f
			at++
		}
	}

	l.Features = l.Features[:at]
}
