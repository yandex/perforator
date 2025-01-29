package mvt

import (
	"reflect"
	"testing"

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
)

func TestLayersClip(t *testing.T) {
	cases := []struct {
		name   string
		bound  orb.Bound
		input  Layers
		output Layers
	}{
		{
			name: "clips polygon and line",
			input: Layers{&Layer{
				Features: []*geojson.Feature{
					geojson.NewFeature(orb.Polygon([]orb.Ring{
						{
							{-10, 10}, {0, 10}, {10, 10}, {10, 5}, {10, -5},
							{10, -10}, {20, -10}, {20, 10}, {40, 10}, {40, 20},
							{20, 20}, {20, 40}, {10, 40}, {10, 20}, {5, 20},
							{-10, 20},
						},
					})),
					geojson.NewFeature(orb.LineString{{-15, 0}, {66, 0}}),
				},
			}},
			output: Layers{&Layer{
				Features: []*geojson.Feature{
					geojson.NewFeature(orb.Polygon([]orb.Ring{
						{
							{0, 10}, {0, 10}, {10, 10}, {10, 5}, {10, 0},
							{20, 0}, {20, 10}, {30, 10}, {30, 20}, {20, 20},
							{20, 30}, {10, 30}, {10, 20}, {5, 20}, {0, 20},
						},
					})),
					geojson.NewFeature(orb.LineString{{0, 0}, {30, 0}}),
				},
			}},
			bound: orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{30, 30}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.input.Clip(tc.bound)
			if !reflect.DeepEqual(tc.input, tc.output) {
				t.Errorf("incorrect clip")
				t.Logf("%v", tc.input)
				t.Logf("%v", tc.output)
			}
		})
	}
}

func TestLayerClip_empty(t *testing.T) {
	layer := &Layer{
		Features: []*geojson.Feature{
			geojson.NewFeature(orb.Polygon{{
				{-1, 1}, {0, 1}, {1, 1}, {1, 5}, {1, -5},
			}}),
			geojson.NewFeature(orb.LineString{{55, 0}, {66, 0}}),
		},
	}

	layer.Clip(orb.Bound{Min: orb.Point{50, -10}, Max: orb.Point{70, 10}})

	if v := len(layer.Features); v != 1 {
		t.Errorf("incorrect number of features: %d", v)
	}

	if v := layer.Features[0].Geometry.GeoJSONType(); v != "LineString" {
		t.Errorf("kept the wrong geometry: %v", layer.Features[0].Geometry)
	}
}
