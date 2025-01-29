# orb/maptile/tilecover [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/maptile/tilecover)

Package `tilecover` computes the covering set of tiles for an `orb.Geometry`.
It is a a port of the nodejs library [tile-cover](https://github.com/mapbox/tile-cover)
which is a port from Google's S2 library. The same set of tests pass.

## Usage

```go
poly := orb.Polygon{}
tiles, err := tilecover.Geometry(poly, zoom)
if err != nil {
	// indicates a non-closed ring
}

for t := range tiles {
    // do something with tile
}

// to merge up to as much as possible to a specific zoom
tiles = tilecover.MergeUp(tiles, 0)
```

## Similar libraries in other languages:

-   [tilecover](https://github.com/mapbox/tile-cover) - Node
