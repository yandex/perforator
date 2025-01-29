# encoding/mvt [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/encoding/mvt)

Package mvt provides functions for encoding and decoding
[Mapbox Vector Tiles](https://www.mapbox.com/vector-tiles/specification/).
The interface is defined as:

```go
type Layer struct {
    Name     string
    Version  uint32
    Extent   uint32
    Features []*geojson.Feature
}

func MarshalGzipped(layers Layers) ([]byte, error)
func Marshal(layers Layers) ([]byte, error)

func UnmarshalGzipped(data []byte) (Layers, error)
func Unmarshal(data []byte) (Layers, error)
```

These function decode the geometry and leave it in the "tile coordinates".
To project it to and from WGS84 (standard lon/lat) use:

```go
func (l Layer) ProjectToTile(tile maptile.Tile)
func (l Layer) ProjectToWGS84(tile maptile.Tile)
```

## Version 1 vs. Version 2

There is no data format difference between v1 and v2. The difference is v2 requires geometries
be simple/clean. e.g. lines that are not self intersecting and polygons that are encoded in the correct winding order.

This library does not do anything to validate the geometry, it just encodes what you give it, so it defaults to v1.
I've seen comments from Mapbox about this and they only want you to claim your library is a v2 encoder if it does cleanup/validation.

However, if you know your geometry is simple/clean you can change the [layer version](https://pkg.go.dev/github.com/paulmach/orb/encoding/mvt#Layer) manually.

## Encoding example

```go
// Start with a set of feature collections defining each layer in lon/lat (WGS84).
collections := map[string]*geojson.FeatureCollection{}

// Convert to a layers object and project to tile coordinates.
layers := mvt.NewLayers(collections)
layers.ProjectToTile(maptile.New(x, y, z))

// In order to be used as source for MapboxGL geometries need to be clipped
// to max allowed extent. (uncomment next line)
// layers.Clip(mvt.MapboxGLDefaultExtentBound)

// Simplify the geometry now that it's in the tile coordinate space.
layers.Simplify(simplify.DouglasPeucker(1.0))

// Depending on use-case remove empty geometry, those two small to be
// represented in this tile space.
// In this case lines shorter than 1, and areas smaller than 1.
layers.RemoveEmpty(1.0, 1.0)

// encoding using the Mapbox Vector Tile protobuf encoding.
data, err := mvt.Marshal(layers) // this data is NOT gzipped.

// Sometimes MVT data is stored and transfered gzip compressed. In that case:
data, err := mvt.MarshalGzipped(layers)
```

## Feature IDs

Since GeoJSON ids can be any number or string they won't necessarily map to vector tile uint64 ids.
This is a common incompatibility between the two types.

During marshaling the code tries to convert the geojson.Feature.ID to a positive integer, possibly parsing a string.
If the number is negative, the id is omitted. If the number is a positive decimal the number is truncated.

For unmarshaling the id will be converted into a float64 to be consistent with how
the encoding/json package decodes numbers.

## Geometry Collections

GeoJSON geometry collections are flattened and their features are encoded individually.
As a result the "collection" information is lost when encoding and there could be more
features in the output (mvt) vs. in the input (GeoJSON)
