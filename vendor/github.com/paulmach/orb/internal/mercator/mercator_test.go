package mercator

import (
	"math"
	"testing"
)

func TestScalarMercator(t *testing.T) {
	x, y := ToPlanar(0, 0, 31)
	lng, lat := ToGeo(x, y, 31)

	if lng != 0.0 {
		t.Errorf("Scalar Mercator, longitude should be 0: %f", lng)
	}

	if lat != 0.0 {
		t.Errorf("Scalar Mercator, latitude should be 0: %f", lat)
	}

	// specific case
	if x, y := ToPlanar(-87.65005229999997, 41.850033, 20); math.Floor(x) != 268988 || math.Floor(y) != 389836 {
		t.Errorf("Scalar Mercator, projection incorrect, got %v %v", x, y)
	}

	if x, y := ToPlanar(-87.65005229999997, 41.850033, 28); math.Floor(x) != 68861112 || math.Floor(y) != 99798110 {
		t.Errorf("Scalar Mercator, projection incorrect, got %v %v", x, y)
	}

	// testing level > 32 to verify correct type conversion
	for _, city := range Cities {
		x, y := ToPlanar(city[1], city[0], 35)
		lng, lat = ToGeo(x, y, 35)

		if math.IsNaN(lng) {
			t.Error("Scalar Mercator, lng is NaN")
		}

		if math.IsNaN(lat) {
			t.Error("Scalar Mercator, lat is NaN")
		}

		if math.Abs(lng-city[1]) > Epsilon {
			t.Errorf("Scalar Mercator, longitude miss match: %f != %f", lng, city[1])
		}

		if math.Abs(lat-city[0]) > Epsilon {
			t.Errorf("Scalar Mercator, latitude miss match: %f != %f", lat, city[0])
		}

	}

	// test polar regions
	if _, y := ToPlanar(0, 89.9, 32); y != (1<<32)-1 {
		t.Errorf("Scalar Mercator, top of the world error, got %v", y)
	}

	if _, y := ToPlanar(0, -89.9, 32); y != 0 {
		t.Errorf("Scalar Mercator, bottom of the world error, got %v", y)
	}
}

func TestToGeoPrecision(t *testing.T) {
	for level := float64(1); level < 35; level++ {
		n := math.Pow(2, level-1)
		// tile with north west coordinate of (0, 0) at each zoom level
		lng, lat := ToGeo(n, n, uint32(level))
		if lng != 0.0 {
			t.Errorf("ToGeo, longitude on level %2.0f should be 0: %f", level, lng)
		}

		if lat != 0.0 {
			t.Errorf("ToGeo, latitude on level %2.0f should be 0: %f", level, lat)
		}
	}
}
