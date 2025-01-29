# encoding/ewkb [![Godoc Reference](https://pkg.go.dev/badge/github.com/paulmach/orb)](https://pkg.go.dev/github.com/paulmach/orb/encoding/ewkb)

This package provides encoding and decoding of [extended WKB](https://en.wikipedia.org/wiki/Well-known_text_representation_of_geometry#Format_variations)
data. This format includes the [SRID](https://en.wikipedia.org/wiki/Spatial_reference_system) in the data.
If the SRID is not needed use the [wkb](../wkb) package for a simpler interface.
The interface is defined as:

```go
func Marshal(geom orb.Geometry, srid int, byteOrder ...binary.ByteOrder) ([]byte, error)
func MarshalToHex(geom orb.Geometry, srid int, byteOrder ...binary.ByteOrder) (string, error)
func MustMarshal(geom orb.Geometry, srid int, byteOrder ...binary.ByteOrder) []byte
func MustMarshalToHex(geom orb.Geometry, srid int, byteOrder ...binary.ByteOrder) string

func NewEncoder(w io.Writer) *Encoder
func (e *Encoder) SetByteOrder(bo binary.ByteOrder) *Encoder
func (e *Encoder) SetSRID(srid int) *Encoder
func (e *Encoder) Encode(geom orb.Geometry) error

func Unmarshal(b []byte) (orb.Geometry, int, error)

func NewDecoder(r io.Reader) *Decoder
func (d *Decoder) Decode() (orb.Geometry, int, error)
```

## Inserting geometry into a database

Depending on the database different formats and functions are supported.

### PostgreSQL and PostGIS

PostGIS stores geometry as EWKB internally. As a result it can be inserted without
a wrapper function.

```go
db.Exec("INSERT INTO geodata(geom) VALUES (ST_GeomFromEWKB($1))", ewkb.Value(coord, 4326))

db.Exec("INSERT INTO geodata(geom) VALUES ($1)", ewkb.Value(coord, 4326))
```

### MySQL/MariaDB

MySQL and MariaDB
[store geometry](https://dev.mysql.com/doc/refman/5.7/en/gis-data-formats.html)
data in WKB format with a 4 byte SRID prefix.

```go
coord := orb.Point{1, 2}

// as WKB in hex format
data := wkb.MustMarshalToHex(coord)
db.Exec("INSERT INTO geodata(geom) VALUES (ST_GeomFromWKB(UNHEX(?), 4326))", data)

// relying on the raw encoding
db.Exec("INSERT INTO geodata(geom) VALUES (?)", ewkb.ValuePrefixSRID(coord, 4326))
```

## Reading geometry from a database query

As stated above, different databases supported different formats and functions.

### PostgreSQL and PostGIS

When working with PostGIS the raw format is EWKB so the wrapper function is not necessary

```go
// both of these queries return the same data
row := db.QueryRow("SELECT ST_AsEWKB(geom) FROM geodata")
row := db.QueryRow("SELECT geom FROM geodata")

// if you don't need the SRID
p := orb.Point{}
err := row.Scan(ewkb.Scanner(&p))
log.Printf("geom: %v", p)

// if you need the SRID
p := orb.Point{}
gs := ewkb.Scanner(&p)
err := row.Scan(gs)

log.Printf("srid: %v", gs.SRID)
log.Printf("geom: %v", gs.Geometry)
log.Printf("also geom: %v", p)
```

### MySQL/MariaDB

```go
// using the ST_AsBinary function
row := db.QueryRow("SELECT st_srid(geom), ST_AsBinary(geom) FROM geodata")
row.Scan(&srid, ewkb.Scanner(&data))

// relying on the raw encoding
row := db.QueryRow("SELECT geom FROM geodata")

// if you don't need the SRID
p := orb.Point{}
err := row.Scan(ewkb.ScannerPrefixSRID(&p))
log.Printf("geom: %v", p)

// if you need the SRID
p := orb.Point{}
gs := ewkb.ScannerPrefixSRID(&p)
err := row.Scan(gs)

log.Printf("srid: %v", gs.SRID)
log.Printf("geom: %v", gs.Geometry)
```
