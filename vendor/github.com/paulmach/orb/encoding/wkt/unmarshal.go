package wkt

import (
	"bytes"
	"errors"
	"regexp"
	"strconv"
	"strings"

	"github.com/paulmach/orb"
)

var (
	// ErrNotWKT is returned when unmarshalling WKT and the data is not valid.
	ErrNotWKT = errors.New("wkt: invalid data")

	// ErrIncorrectGeometry is returned when unmarshalling WKT data into the wrong type.
	// For example, unmarshaling linestring data into a point.
	ErrIncorrectGeometry = errors.New("wkt: incorrect geometry")

	// ErrUnsupportedGeometry is returned when geometry type is not supported by this lib.
	ErrUnsupportedGeometry = errors.New("wkt: unsupported geometry")

	doubleParen = regexp.MustCompile(`\)[\s|\t]*\)([\s|\t]*,[\s|\t]*)\([\s|\t]*\(`)
	singleParen = regexp.MustCompile(`\)([\s|\t]*,[\s|\t]*)\(`)
)

// UnmarshalPoint returns the point represented by the wkt string.
// Will return ErrIncorrectGeometry if the wkt is not a point.
func UnmarshalPoint(s string) (orb.Point, error) {
	s = trimSpace(s)
	prefix := upperPrefix(s)
	if !bytes.HasPrefix(prefix, []byte("POINT")) {
		return orb.Point{}, ErrIncorrectGeometry
	}

	return unmarshalPoint(s)
}

func unmarshalPoint(s string) (orb.Point, error) {
	s, err := trimSpaceBrackets(s[5:])
	if err != nil {
		return orb.Point{}, err
	}

	tp, err := parsePoint(s)
	if err != nil {
		return orb.Point{}, err
	}

	return tp, nil
}

// parsePoint pase point by (x y)
func parsePoint(s string) (p orb.Point, err error) {
	one, two, ok := cut(s, " ")
	if !ok {
		return orb.Point{}, ErrNotWKT
	}

	x, err := strconv.ParseFloat(one, 64)
	if err != nil {
		return orb.Point{}, ErrNotWKT
	}

	y, err := strconv.ParseFloat(two, 64)
	if err != nil {
		return orb.Point{}, ErrNotWKT
	}

	return orb.Point{x, y}, nil
}

// UnmarshalMultiPoint returns the multi-point represented by the wkt string.
// Will return ErrIncorrectGeometry if the wkt is not a multi-point.
func UnmarshalMultiPoint(s string) (orb.MultiPoint, error) {
	s = trimSpace(s)
	prefix := upperPrefix(s)
	if !bytes.HasPrefix(prefix, []byte("MULTIPOINT")) {
		return nil, ErrIncorrectGeometry
	}

	return unmarshalMultiPoint(s)
}

func unmarshalMultiPoint(s string) (orb.MultiPoint, error) {
	if strings.EqualFold(s, "MULTIPOINT EMPTY") {
		return orb.MultiPoint{}, nil
	}

	s, err := trimSpaceBrackets(s[10:])
	if err != nil {
		return nil, err
	}

	count := strings.Count(s, ",")
	mp := make(orb.MultiPoint, 0, count+1)

	err = splitOnComma(s, func(p string) error {
		p, err := trimSpaceBrackets(p)
		if err != nil {
			return err
		}

		tp, err := parsePoint(p)
		if err != nil {
			return err
		}

		mp = append(mp, tp)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return mp, nil
}

// UnmarshalLineString returns the linestring represented by the wkt string.
// Will return ErrIncorrectGeometry if the wkt is not a linestring.
func UnmarshalLineString(s string) (orb.LineString, error) {
	s = trimSpace(s)
	prefix := upperPrefix(s)
	if !bytes.HasPrefix(prefix, []byte("LINESTRING")) {
		return nil, ErrIncorrectGeometry
	}

	return unmarshalLineString(s)
}

func unmarshalLineString(s string) (orb.LineString, error) {
	if strings.EqualFold(s, "LINESTRING EMPTY") {
		return orb.LineString{}, nil
	}

	s, err := trimSpaceBrackets(s[10:])
	if err != nil {
		return nil, err
	}

	count := strings.Count(s, ",")
	ls := make(orb.LineString, 0, count+1)

	err = splitOnComma(s, func(p string) error {
		tp, err := parsePoint(p)
		if err != nil {
			return err
		}

		ls = append(ls, tp)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return ls, nil
}

// UnmarshalMultiLineString returns the multi-linestring represented by the wkt string.
// Will return ErrIncorrectGeometry if the wkt is not a multi-linestring.
func UnmarshalMultiLineString(s string) (orb.MultiLineString, error) {
	s = trimSpace(s)
	prefix := upperPrefix(s)
	if !bytes.HasPrefix(prefix, []byte("MULTILINESTRING")) {
		return nil, ErrIncorrectGeometry
	}

	return unmarshalMultiLineString(s)
}

func unmarshalMultiLineString(s string) (orb.MultiLineString, error) {
	if strings.EqualFold(s, "MULTILINESTRING EMPTY") {
		return orb.MultiLineString{}, nil
	}

	s, err := trimSpaceBrackets(s[15:])
	if err != nil {
		return nil, err
	}

	var tmls orb.MultiLineString
	err = splitByRegexpYield(
		s,
		singleParen,
		func(i int) {
			tmls = make(orb.MultiLineString, 0, i)
		},
		func(ls string) error {
			ls, err := trimSpaceBrackets(ls)
			if err != nil {
				return err
			}

			count := strings.Count(ls, ",")
			tls := make(orb.LineString, 0, count+1)

			err = splitOnComma(ls, func(p string) error {
				tp, err := parsePoint(p)
				if err != nil {
					return err
				}

				tls = append(tls, tp)
				return nil
			})
			if err != nil {
				return err
			}

			tmls = append(tmls, tls)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return tmls, nil
}

// UnmarshalPolygon returns the polygon represented by the wkt string.
// Will return ErrIncorrectGeometry if the wkt is not a polygon.
func UnmarshalPolygon(s string) (orb.Polygon, error) {
	s = trimSpace(s)
	prefix := upperPrefix(s)
	if !bytes.HasPrefix(prefix, []byte("POLYGON")) {
		return nil, ErrIncorrectGeometry
	}

	return unmarshalPolygon(s)
}

func unmarshalPolygon(s string) (orb.Polygon, error) {
	if strings.EqualFold(s, "POLYGON EMPTY") {
		return orb.Polygon{}, nil
	}

	s, err := trimSpaceBrackets(s[7:])
	if err != nil {
		return nil, err
	}

	var poly orb.Polygon
	err = splitByRegexpYield(
		s,
		singleParen,
		func(i int) {
			poly = make(orb.Polygon, 0, i)
		},
		func(r string) error {
			r, err := trimSpaceBrackets(r)
			if err != nil {
				return err
			}

			count := strings.Count(r, ",")
			ring := make(orb.Ring, 0, count+1)

			err = splitOnComma(r, func(p string) error {
				tp, err := parsePoint(p)
				if err != nil {
					return err
				}
				ring = append(ring, tp)
				return nil
			})
			if err != nil {
				return err
			}

			poly = append(poly, ring)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return poly, nil
}

// UnmarshalMultiPolygon returns the multi-polygon represented by the wkt string.
// Will return ErrIncorrectGeometry if the wkt is not a multi-polygon.
func UnmarshalMultiPolygon(s string) (orb.MultiPolygon, error) {
	s = trimSpace(s)
	prefix := upperPrefix(s)
	if !bytes.HasPrefix(prefix, []byte("MULTIPOLYGON")) {
		return nil, ErrIncorrectGeometry
	}

	return unmarshalMultiPolygon(s)
}

func unmarshalMultiPolygon(s string) (orb.MultiPolygon, error) {
	if strings.EqualFold(s, "MULTIPOLYGON EMPTY") {
		return orb.MultiPolygon{}, nil
	}

	s, err := trimSpaceBrackets(s[12:])
	if err != nil {
		return nil, err
	}

	var mpoly orb.MultiPolygon
	err = splitByRegexpYield(
		s,
		doubleParen,
		func(i int) {
			mpoly = make(orb.MultiPolygon, 0, i)
		},
		func(poly string) error {
			poly, err := trimSpaceBrackets(poly)
			if err != nil {
				return err
			}

			var tpoly orb.Polygon
			err = splitByRegexpYield(
				poly,
				singleParen,
				func(i int) {
					tpoly = make(orb.Polygon, 0, i)
				},
				func(r string) error {
					r, err := trimSpaceBrackets(r)
					if err != nil {
						return err
					}

					count := strings.Count(r, ",")
					tr := make(orb.Ring, 0, count+1)

					err = splitOnComma(r, func(s string) error {
						tp, err := parsePoint(s)
						if err != nil {
							return err
						}

						tr = append(tr, tp)
						return nil
					})
					if err != nil {
						return err
					}

					tpoly = append(tpoly, tr)
					return nil
				},
			)
			if err != nil {
				return err
			}

			mpoly = append(mpoly, tpoly)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}

	return mpoly, nil
}

// UnmarshalCollection returns the geometry collection represented by the wkt string.
// Will return ErrIncorrectGeometry if the wkt is not a geometry collection.
func UnmarshalCollection(s string) (orb.Collection, error) {
	s = trimSpace(s)
	prefix := upperPrefix(s)
	if !bytes.HasPrefix(prefix, []byte("GEOMETRYCOLLECTION")) {
		return nil, ErrIncorrectGeometry
	}

	return unmarshalCollection(s)
}

func unmarshalCollection(s string) (orb.Collection, error) {
	if strings.EqualFold(s, "GEOMETRYCOLLECTION EMPTY") {
		return orb.Collection{}, nil
	}

	if len(s) == 18 { // just GEOMETRYCOLLECTION
		return nil, ErrNotWKT
	}

	geometries := splitGeometryCollection(s[18:])
	if len(geometries) == 0 {
		return orb.Collection{}, nil
	}

	c := make(orb.Collection, 0, len(geometries))
	for _, g := range geometries {
		if len(g) == 0 {
			continue
		}

		tg, err := Unmarshal(g)
		if err != nil {
			return nil, err
		}

		c = append(c, tg)
	}

	return c, nil
}

// splitGeometryCollection split GEOMETRYCOLLECTION to more geometry
func splitGeometryCollection(s string) (r []string) {
	r = make([]string, 0)
	stack := make([]rune, 0)
	l := len(s)
	for i, v := range s {
		if !strings.Contains(string(stack), "(") {
			stack = append(stack, v)
			continue
		}
		if ('A' <= v && v < 'Z') || ('a' <= v && v < 'z') {
			t := string(stack)
			r = append(r, t[:len(t)-1])
			stack = make([]rune, 0)
			stack = append(stack, v)
			continue
		}
		if i == l-1 {
			r = append(r, string(stack))
			continue
		}
		stack = append(stack, v)
	}
	return
}

// Unmarshal return a geometry by parsing the WKT string.
func Unmarshal(s string) (orb.Geometry, error) {
	var (
		g   orb.Geometry
		err error
	)

	s = trimSpace(s)
	prefix := upperPrefix(s)

	if bytes.HasPrefix(prefix, []byte("POINT")) {
		g, err = unmarshalPoint(s)
	} else if bytes.HasPrefix(prefix, []byte("LINESTRING")) {
		g, err = unmarshalLineString(s)
	} else if bytes.HasPrefix(prefix, []byte("POLYGON")) {
		g, err = unmarshalPolygon(s)
	} else if bytes.HasPrefix(prefix, []byte("MULTIPOINT")) {
		g, err = unmarshalMultiPoint(s)
	} else if bytes.HasPrefix(prefix, []byte("MULTILINESTRING")) {
		g, err = unmarshalMultiLineString(s)
	} else if bytes.HasPrefix(prefix, []byte("MULTIPOLYGON")) {
		g, err = unmarshalMultiPolygon(s)
	} else if bytes.HasPrefix(prefix, []byte("GEOMETRYCOLLECTION")) {
		g, err = unmarshalCollection(s)
	} else {
		return nil, ErrUnsupportedGeometry
	}

	if err != nil {
		return nil, err
	}

	return g, nil
}

// splitByRegexpYield splits the input by the regexp. The first callback can
// be used to initialize an array with the size of the result, the second
// is the callback with the matches.
// We use a yield function because it was faster/used less memory than
// allocating an array of the results.
func splitByRegexpYield(s string, re *regexp.Regexp, set func(int), yield func(string) error) error {
	indexes := re.FindAllStringSubmatchIndex(s, -1)
	set(len(indexes) + 1)
	start := 0
	for _, element := range indexes {
		err := yield(s[start:element[2]])
		if err != nil {
			return err
		}
		start = element[3]
	}

	return yield(s[start:])
}

// splitOnComma is optimized to split on the regex [\s|\t|\n]*,[\s|\t|\n]*
// i.e. comma with possible spaces on each side. e.g. '  ,  '
// We use a yield function because it was faster/used less memory than
// allocating an array of the results.
func splitOnComma(s string, yield func(s string) error) error {
	// in WKT points are separtated by commas, coordinates in points are separted by spaces
	// e.g. 1 2,3 4,5 6,7 81 2,5 4
	// we want to split this and find each point.

	// at is right after the previous space-comma-space match.
	// once a space-comma-space match is found, we go from 'at' to the start
	// of the match, that's the split that needs to be returned.
	var at int

	var start int // the start of a space-comma-space section

	// a space starts a section, we need to see a comma for it to be a valid section
	var sawSpace, sawComma bool
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			if !sawSpace {
				sawSpace = true
				start = i
			}
			sawComma = true
			continue
		}

		if v := s[i]; v == ' ' || v == '\t' || v == '\n' {
			if !sawSpace {
				sawSpace = true
				start = i
			}
			continue
		}

		if sawComma {
			err := yield(s[at:start])
			if err != nil {
				return err
			}
			at = i
		}
		sawSpace = false
		sawComma = false
	}

	return yield(s[at:])
}

// trimSpaceBrackets trim space and brackets
func trimSpaceBrackets(s string) (string, error) {
	s = trimSpace(s)
	if len(s) == 0 {
		return s, nil
	}

	if s[0] == '(' {
		s = s[1:]
	} else {
		return "", ErrNotWKT
	}

	if s[len(s)-1] == ')' {
		s = s[:len(s)-1]
	} else {
		return "", ErrNotWKT
	}

	return trimSpace(s), nil
}

func trimSpace(s string) string {
	if len(s) == 0 {
		return s
	}

	var start, end int

	for start = 0; start < len(s); start++ {
		if v := s[start]; v != ' ' && v != '\t' && v != '\n' {
			break
		}
	}

	for end = len(s) - 1; end >= 0; end-- {
		if v := s[end]; v != ' ' && v != '\t' && v != '\n' {
			break
		}
	}

	if start >= end {
		return ""
	}

	return s[start : end+1]
}

// gets the ToUpper case of the first 20 chars.
// This is to determin the type without doing a full strings.ToUpper
func upperPrefix(s string) []byte {
	prefix := make([]byte, 20)
	for i := 0; i < 20 && i < len(s); i++ {
		if 'a' <= s[i] && s[i] <= 'z' {
			prefix[i] = s[i] - ('a' - 'A')
		} else {
			prefix[i] = s[i]
		}
	}

	return prefix
}

// coppied here from strings.Cut so we don't require go1.18
func cut(s, sep string) (before, after string, found bool) {
	if i := strings.Index(s, sep); i >= 0 {
		return s[:i], s[i+len(sep):], true
	}
	return s, "", false
}
