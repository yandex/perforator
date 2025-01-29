# orb/geojson [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/geojson)

This package **encodes and decodes** [GeoJSON](http://geojson.org/) into Go structs
using the geometries in the [orb](https://github.com/paulmach/orb) package.

Supports both the [json.Marshaler](https://pkg.go.dev/encoding/json#Marshaler) and
[json.Unmarshaler](https://pkg.go.dev/encoding/json#Unmarshaler) interfaces.
The package also provides helper functions such as `UnmarshalFeatureCollection` and `UnmarshalFeature`.

The types also support BSON via the [bson.Marshaler](https://pkg.go.dev/go.mongodb.org/mongo-driver/bson#Marshaler) and
[bson.Unmarshaler](https://pkg.go.dev/go.mongodb.org/mongo-driver/bson#Unmarshaler) interfaces.
These types can be used directly when working with MongoDB.

## Unmarshalling (JSON -> Go)

```go
rawJSON := []byte(`
  { "type": "FeatureCollection",
    "features": [
      { "type": "Feature",
        "geometry": {"type": "Point", "coordinates": [102.0, 0.5]},
        "properties": {"prop0": "value0"}
      }
    ]
  }`)

fc, _ := geojson.UnmarshalFeatureCollection(rawJSON)

// or

fc := geojson.NewFeatureCollection()
err := json.Unmarshal(rawJSON, &fc)

// Geometry will be unmarshalled into the correct geo.Geometry type.
point := fc.Features[0].Geometry.(orb.Point)
```

## Marshalling (Go -> JSON)

```go
fc := geojson.NewFeatureCollection()
fc.Append(geojson.NewFeature(orb.Point{1, 2}))

rawJSON, _ := fc.MarshalJSON()

// or
blob, _ := json.Marshal(fc)
```

## Foreign/extra members in a feature collection

```go
rawJSON := []byte(`
  { "type": "FeatureCollection",
    "generator": "myapp",
    "timestamp": "2020-06-15T01:02:03Z",
    "features": [
      { "type": "Feature",
        "geometry": {"type": "Point", "coordinates": [102.0, 0.5]},
        "properties": {"prop0": "value0"}
      }
    ]
  }`)

fc, _ := geojson.UnmarshalFeatureCollection(rawJSON)

fc.ExtraMembers["generator"] // == "myApp"
fc.ExtraMembers["timestamp"] // == "2020-06-15T01:02:03Z"

// Marshalling will include values in `ExtraMembers` in the
// base featureCollection object.
```

## Performance

For performance critical applications, consider a
third party replacement of "encoding/json" like [github.com/json-iterator/go](https://github.com/json-iterator/go)

This can be enabled with something like this:

```go
import (
  jsoniter "github.com/json-iterator/go"
  "github.com/paulmach/orb"
)

var c = jsoniter.Config{
  EscapeHTML:              true,
  SortMapKeys:             false,
  MarshalFloatWith6Digits: true,
}.Froze()

CustomJSONMarshaler = c
CustomJSONUnmarshaler = c
```

The above change can have dramatic performance implications, see the benchmarks below
on a 100k feature collection file:

```
benchmark                             old ns/op     new ns/op     delta
BenchmarkFeatureMarshalJSON-12        2694543       733480        -72.78%
BenchmarkFeatureUnmarshalJSON-12      5383825       2738183       -49.14%
BenchmarkGeometryMarshalJSON-12       210107        62789         -70.12%
BenchmarkGeometryUnmarshalJSON-12     691472        144689        -79.08%

benchmark                             old allocs     new allocs     delta
BenchmarkFeatureMarshalJSON-12        7818           2316           -70.38%
BenchmarkFeatureUnmarshalJSON-12      23047          31946          +38.61%
BenchmarkGeometryMarshalJSON-12       2              3              +50.00%
BenchmarkGeometryUnmarshalJSON-12     2042           18             -99.12%

benchmark                             old bytes     new bytes     delta
BenchmarkFeatureMarshalJSON-12        794088        490251        -38.26%
BenchmarkFeatureUnmarshalJSON-12      766354        1068497       +39.43%
BenchmarkGeometryMarshalJSON-12       24787         18650         -24.76%
BenchmarkGeometryUnmarshalJSON-12     79784         51374         -35.61%
```

## Feature Properties

GeoJSON features can have properties of any type. This can cause issues in a statically typed
language such as Go. Included is a `Properties` type with some helper methods that will try to
force convert a property. An optional default, will be used if the property is missing or the wrong
type.

```go
f.Properties.MustBool(key string, def ...bool) bool
f.Properties.MustFloat64(key string, def ...float64) float64
f.Properties.MustInt(key string, def ...int) int
f.Properties.MustString(key string, def ...string) string
```
