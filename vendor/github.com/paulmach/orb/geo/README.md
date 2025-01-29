# orb/geo [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/geo)

The geometries defined in the `orb` package are generic 2d geometries.
Depending on what projection they're in, e.g. lon/lat or flat on the plane,
area and distance calculations are different. This package implements methods
that assume the lon/lat or WGS84 projection.

## Examples

Area of the [San Francisco Main Library](https://www.openstreetmap.org/way/24446086):

```go
poly := orb.Polygon{
    {
        { -122.4163816, 37.7792782 },
        { -122.4162786, 37.7787626 },
        { -122.4151027, 37.7789118 },
        { -122.4152143, 37.7794274 },
        { -122.4163816, 37.7792782 },
    },
}

a := geo.Area(poly)

fmt.Printf("%f m^2", a)
// Output:
// 6073.368008 m^2
```

Distance between two points:

```go
oakland := orb.Point{-122.270833, 37.804444}
sf := orb.Point{-122.416667, 37.783333}

d := geo.Distance(oakland, sf)

fmt.Printf("%0.3f meters", d)
// Output:
// 13042.047 meters
```

Circumference of the [San Francisco Main Library](https://www.openstreetmap.org/way/24446086):

```go
poly := orb.Polygon{
    {
        { -122.4163816, 37.7792782 },
        { -122.4162786, 37.7787626 },
        { -122.4151027, 37.7789118 },
        { -122.4152143, 37.7794274 },
        { -122.4163816, 37.7792782 },
    },
}
l := geo.Length(poly)

fmt.Printf("%0.0f meters", l)
// Output:
// 325 meters
```
