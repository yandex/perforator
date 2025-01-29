# orb/clip [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/clip)

Package orb/clip provides functions for clipping lines and polygons to a bounding box.

-   uses [Cohen-Sutherland algorithm](https://en.wikipedia.org/wiki/Cohen%E2%80%93Sutherland_algorithm) for line clipping
-   uses [Sutherland-Hodgman algorithm](https://en.wikipedia.org/wiki/Sutherland%E2%80%93Hodgman_algorithm) for polygon clipping

## Example

```go
bound := orb.Bound{Min: orb.Point{0, 0}, Max: orb.Point{30, 30}}

ls := orb.LineString{
    {-10, 10}, {10, 10}, {10, -10}, {20, -10}, {20, 10},
    {40, 10}, {40, 20}, {20, 20}, {20, 40}, {10, 40},
    {10, 20}, {5, 20}, {-10, 20},
}

// works on and returns an orb.Geometry interface.
clipped = clip.Geometry(bound, ls)

// or clip the line string directly
clipped = clip.LineString(bound, ls)
```

## List of sub-package utilities

-   [`smartclip`](smartclip) - handles partial 2d geometries

## Acknowledgements

This library is based on [mapbox/lineclip](https://github.com/mapbox/lineclip).
