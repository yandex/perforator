# orb/resample [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/resample)

Package `resample` has a couple functions for resampling line geometry
into more or less evenly spaces points.

```go
func Resample(ls orb.LineString, df orb.DistanceFunc, totalPoints int) orb.LineString
func ToInterval(ls orb.LineString, df orb.DistanceFunc, dist float64) orb.LineString
```

For example, resampling a line string so the points are 1 planar unit apart:

```go
ls := resample.ToInterval(ls, planar.Distance, 1.0)
```
