# orb/clip/smartclip [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/clip/smartclip)

This package extends the clip functionality to handle partial 2d geometries. The input polygon
rings need to only intersect the bound. The algorithm will use that, plus the orientation, to
wrap/close the rings around the edge of the bound.

The use case is [OSM multipolyon relations](https://wiki.openstreetmap.org/wiki/Relation#Multipolygon)
where a ring (inner or outer) contains multiple ways but only one is in the current viewport.
With only the ways intersecting the viewport and their orientation the correct shape can be drawn.

## Example

```go
bound := orb.Bound{Min: orb.Point{1, 1}, Max: orb.Point{10, 10}}

// a partial ring cutting the bound down the middle.
ring := orb.Ring{{0, 0}, {11, 11}}
clipped := smartclip.Ring(bound, ring, orb.CCW)

// clipped is a multipolyon with one ring that wraps counter-clockwise
// around the top triangle of the box
// [[[[1 1] [10 10] [5.5 10] [1 10] [1 5.5] [1 1]]]]
```
