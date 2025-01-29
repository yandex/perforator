# orb/simplify [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/simplify)

This package implements several reducing/simplifing function for `orb.Geometry` types.

Currently implemented:

-   [Douglas-Peucker](#dp)
-   [Visvalingam](#vis)
-   [Radial](#radial)

**Note:** The geometry object CAN be modified, use `Clone()` if a copy is required.

## <a name="dp"></a>Douglas-Peucker

Probably the most popular simplification algorithm. For algorithm details, see
[wikipedia](http://en.wikipedia.org/wiki/Ramer%E2%80%93Douglas%E2%80%93Peucker_algorithm).

The algorithm is a pass through for 1d geometry, e.g. Point and MultiPoint.
The algorithms can modify the original geometry, use `Clone()` if a copy is required.

Usage:

    original := orb.LineString{}
    reduced := simplify.DouglasPeucker(threshold).Simplify(original.Clone())

## <a name="vis"></a>Visvalingam

See Mike Bostock's explanation for
[algorithm details](http://bost.ocks.org/mike/simplify/).

The algorithm is a pass through for 1d geometry, e.g. Point and MultiPoint.
The algorithms can modify the original geometry, use `Clone()` if a copy is required.

Usage:

```go
original := orb.Ring{}

// will remove all whose triangle is smaller than `threshold`
reduced := simplify.VisvalingamThreshold(threshold).Simplify(original)

// will remove points until there are only `toKeep` points left.
reduced := simplify.VisvalingamKeep(toKeep).Simplify(original)

// One can also combine the parameters.
// This will continue to remove points until:
//  - there are no more below the threshold,
//  - or the new path is of length `toKeep`
reduced := simplify.Visvalingam(threshold, toKeep).Simplify(original)
```

## <a name="radial"></a>Radial

Radial reduces the path by removing points that are close together.
A full [algorithm description](http://psimpl.sourceforge.net/radial-distance.html).

The algorithm is a pass through for 1d geometry, like Point and MultiPoint.
The algorithms can modify the original geometry, use `Clone()` if a copy is required.

Usage:

```go
original := geo.Polygon{}

// this method uses a Euclidean distance measure.
reduced := simplify.Radial(planar.Distance, threshold).Simplify(path)

// if the points are in the lng/lat space Radial Geo will
// compute the geo distance between the coordinates.
reduced:= simplify.Radial(geo.Distance, meters).Simplify(path)
```
