package render

import (
	"image/color"
	"math"
)

// From https://en.wikipedia.org/wiki/HSL_and_HSV#HSV_to_RGB_alternative
func HSV(h, s, v float64) color.RGBA {
	f := func(n int) uint8 {
		k := math.Mod(float64(n)+h/60.0, 6.0)
		c := v - v*s*max(0.0, min(k, 4.0-k, 1.0))
		return uint8(math.Round(c * 255))
	}

	return color.RGBA{
		R: f(5),
		G: f(3),
		B: f(1),
	}
}
