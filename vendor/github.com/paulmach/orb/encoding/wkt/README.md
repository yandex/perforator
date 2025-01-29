# encoding/wkt [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/encoding/wkt)

This package provides encoding and decoding of [WKT](https://en.wikipedia.org/wiki/Well-known_text_representation_of_geometry)
data. The interface is defined as:

```go
func MarshalString(orb.Geometry) string

func Unmarshal(string) (orb.Geometry, error)
func UnmarshalPoint(string) (orb.Point, err error)
func UnmarshalMultiPoint(string) (orb.MultiPoint, err error)
func UnmarshalLineString(string) (orb.LineString, err error)
func UnmarshalMultiLineString(string) (orb.MultiLineString, err error)
func UnmarshalPolygon(string) (orb.Polygon, err error)
func UnmarshalMultiPolygon(string) (orb.MultiPolygon, err error)
func UnmarshalCollection(string) (orb.Collection, err error)
```
