# orb/planar [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/planar)

The geometries defined in the `orb` package are generic 2d geometries.
Depending on what projection they're in, e.g. lon/lat or flat on the plane,
area and distance calculations are different. This package implements methods
that assume the planar or Euclidean context.

## Examples

Area of 3-4-5 triangle:

```go
r := orb.Ring{{0, 0}, {3, 0}, {0, 4}, {0, 0}}
a := planar.Area(r)

fmt.Println(a)
// Output:
// 6
```

Distance between two points:

```go
d := planar.Distance(orb.Point{0, 0}, orb.Point{3, 4})

fmt.Println(d)
// Output:
// 5
```

Length/circumference of a 3-4-5 triangle:

```go
r := orb.Ring{{0, 0}, {3, 0}, {0, 4}, {0, 0}}
l := planar.Length(r)

fmt.Println(l)
// Output:
// 12
```
