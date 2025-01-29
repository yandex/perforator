# orb/maptile [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/maptile)

Package `maptile` provides types and methods for working with
[web mercator map tiles](https://www.google.com/search?q=web+mercator+map+tiles).
It defines a tile as:

```go
type Tile struct {
    X, Y uint32
    Z    Zoom
}

type Zoom uint32
```

Functions are provided to create tiles from lon/lat points as well as
[quadkeys](https://msdn.microsoft.com/en-us/library/bb259689.aspx).
The tile defines helper methods such as `Parent()`, `Children()`, `Siblings()`, etc.

## List of sub-package utilities

-   [`tilecover`](tilecover) - computes the covering set of tiles for an `orb.Geometry`.

## Similar libraries in other languages:

-   [mercantile](https://github.com/mapbox/mercantile) - Python
-   [sphericalmercator](https://github.com/mapbox/sphericalmercator) - Node
-   [tilebelt](https://github.com/mapbox/tilebelt) - Node
