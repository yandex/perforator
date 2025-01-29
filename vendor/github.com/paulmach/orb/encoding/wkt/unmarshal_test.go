package wkt

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/paulmach/orb"
)

func TestTrimSpaceBrackets(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		expected string
	}{
		{
			name:     "empty string",
			s:        "",
			expected: "",
		},
		{
			name:     "blank string",
			s:        "   ",
			expected: "",
		},
		{
			name:     "single point",
			s:        "(1 2)",
			expected: "1 2",
		},
		{
			name:     "double brackets",
			s:        "((1 2),(0.5 1.5))",
			expected: "(1 2),(0.5 1.5)",
		},
		{
			name:     "multiple values",
			s:        "(1 2,0.5 1.5)",
			expected: "1 2,0.5 1.5",
		},
		{
			name:     "multiple points",
			s:        "((1 2,3 4),(5 6,7 8))",
			expected: "(1 2,3 4),(5 6,7 8)",
		},
		{
			name:     "triple brackets",
			s:        "(((1 2,3 4)),((5 6,7 8),(1 2,5 4)))",
			expected: "((1 2,3 4)),((5 6,7 8),(1 2,5 4))",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := trimSpaceBrackets(tc.s)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if v != tc.expected {
				t.Log(trimSpaceBrackets(tc.s))
				t.Log(tc.expected)
				t.Errorf("trim space and brackets error")
			}
		})
	}
}

func TestTrimSpaceBrackets_errors(t *testing.T) {
	cases := []struct {
		name string
		s    string
		err  error
	}{
		{
			name: "no brackets",
			s:    "1 2",
			err:  ErrNotWKT,
		},
		{
			name: "no start bracket",
			s:    "1 2)",
			err:  ErrNotWKT,
		},
		{
			name: "no end bracket",
			s:    "(1 2",
			err:  ErrNotWKT,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := trimSpaceBrackets(tc.s)
			if err != tc.err {
				t.Fatalf("wrong error: %v != %v", err, tc.err)
			}
		})
	}
}

func TestUnmarshalPoint(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		expected orb.Point
	}{
		{
			name:     "int",
			s:        "POINT(1 2)",
			expected: orb.Point{1, 2},
		},
		{
			name:     "float64",
			s:        "POINT(1.34 2.35)",
			expected: orb.Point{1.34, 2.35},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// you know it's a point
			p, err := UnmarshalPoint(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			if !p.Equal(tc.expected) {
				t.Log(p)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// lower case
			p, err = UnmarshalPoint("  " + strings.ToLower(tc.s))
			if err != nil {
				t.Fatal(err)
			}

			if !p.Equal(tc.expected) {
				t.Log(p)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// via generic unmarshal
			geom, err := Unmarshal(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			p = geom.(orb.Point)

			if !p.Equal(tc.expected) {
				t.Log(p)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}
		})
	}
}

func TestUnmarshalPoint_errors(t *testing.T) {
	cases := []struct {
		name string
		s    string
		err  error
	}{
		{
			name: "just name",
			s:    "POINT",
			err:  ErrNotWKT,
		},
		{
			name: "too many points",
			s:    "POINT(1.34 2.35 3.36)",
			err:  ErrNotWKT,
		},
		{
			name: "not a point",
			s:    "MULTIPOINT((1.34 2.35))",
			err:  ErrIncorrectGeometry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UnmarshalPoint(tc.s)
			if err != tc.err {
				t.Fatalf("incorrect error: %v != %v", err, tc.err)
			}
		})
	}
}

func TestUnmarshalMultiPoint(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		expected orb.MultiPoint
	}{
		{
			name:     "empty",
			s:        "MULTIPOINT EMPTY",
			expected: orb.MultiPoint{},
		},
		{
			name:     "1 point",
			s:        "MULTIPOINT((1 2))",
			expected: orb.MultiPoint{{1, 2}},
		},
		{
			name:     "2 points",
			s:        "MULTIPOINT((1 2),(0.5 1.5))",
			expected: orb.MultiPoint{{1, 2}, {0.5, 1.5}},
		},
		{
			name:     "spaces",
			s:        "MULTIPOINT((1 2)  ,	(0.5 1.5))",
			expected: orb.MultiPoint{{1, 2}, {0.5, 1.5}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// you know it's multipoint
			mp, err := UnmarshalMultiPoint(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			if !mp.Equal(tc.expected) {
				t.Log(mp)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// lower case
			mp, err = UnmarshalMultiPoint("  " + strings.ToLower(tc.s))
			if err != nil {
				t.Fatal(err)
			}

			if !mp.Equal(tc.expected) {
				t.Log(mp)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// via generic unmarshall
			geom, err := Unmarshal(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			mp = geom.(orb.MultiPoint)

			if !mp.Equal(tc.expected) {
				t.Log(mp)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}
		})
	}
}

func TestUnmarshalMultiPoint_errors(t *testing.T) {
	cases := []struct {
		name string
		s    string
		err  error
	}{
		{
			name: "just name",
			s:    "MULTIPOINT",
			err:  ErrNotWKT,
		},
		{
			name: "too many points",
			s:    "MULTIPOINT((1 2),(3 4 5))",
			err:  ErrNotWKT,
		},
		{
			name: "not a multipoint",
			s:    "POINT(1 2)",
			err:  ErrIncorrectGeometry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UnmarshalMultiPoint(tc.s)
			if err != tc.err {
				t.Fatalf("incorrect error: %v != %v", err, tc.err)
			}
		})
	}
}

func TestUnmarshalLineString(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		expected orb.LineString
	}{
		{
			name:     "empty",
			s:        "LINESTRING EMPTY",
			expected: orb.LineString{},
		},
		{
			name:     "2 points",
			s:        "LINESTRING(1 2,0.5 1.5)",
			expected: orb.LineString{{1, 2}, {0.5, 1.5}},
		},
		{
			name:     "spaces",
			s:        "LINESTRING(1 2 , 0.5 1.5)",
			expected: orb.LineString{{1, 2}, {0.5, 1.5}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// you know it's a linestring
			ls, err := UnmarshalLineString(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			if !ls.Equal(tc.expected) {
				t.Log(ls)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// lower case
			ls, err = UnmarshalLineString("  " + strings.ToLower(tc.s))
			if err != nil {
				t.Fatal(err)
			}

			if !ls.Equal(tc.expected) {
				t.Log(ls)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// via generic unmarshal
			geom, err := Unmarshal(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			ls = geom.(orb.LineString)

			if !ls.Equal(tc.expected) {
				t.Log(ls)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}
		})
	}
}

func TestUnmarshalLineString_errors(t *testing.T) {
	cases := []struct {
		name string
		s    string
		err  error
	}{
		{
			name: "just name",
			s:    "LINESTRING",
			err:  ErrNotWKT,
		},
		{
			name: "too many points",
			s:    "LINESTRING(1 2,3 4 5)",
			err:  ErrNotWKT,
		},
		{
			name: "not a multipoint",
			s:    "POINT(1 2)",
			err:  ErrIncorrectGeometry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UnmarshalLineString(tc.s)
			if err != tc.err {
				t.Fatalf("incorrect error: %v != %v", err, tc.err)
			}
		})
	}
}

func TestUnmarshalMultiLineString(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		expected orb.MultiLineString
	}{
		{
			name:     "empty",
			s:        "MULTILINESTRING EMPTY",
			expected: orb.MultiLineString{},
		},
		{
			name:     "2 lines",
			s:        "MULTILINESTRING((1 2,3 4),(5 6,7 8))",
			expected: orb.MultiLineString{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// you know it's a linestring
			mls, err := UnmarshalMultiLineString(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			if !mls.Equal(tc.expected) {
				t.Log(mls)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// lower case
			mls, err = UnmarshalMultiLineString("  " + strings.ToLower(tc.s))
			if err != nil {
				t.Fatal(err)
			}

			if !mls.Equal(tc.expected) {
				t.Log(mls)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// via generic unmarshal
			geom, err := Unmarshal(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			mls = geom.(orb.MultiLineString)

			if !mls.Equal(tc.expected) {
				t.Log(mls)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}
		})
	}
}

func TestUnmarshalMultiLineString_errors(t *testing.T) {
	cases := []struct {
		name string
		s    string
		err  error
	}{
		{
			name: "just name",
			s:    "MULTILINESTRING",
			err:  ErrNotWKT,
		},
		{
			name: "too many points",
			s:    "MULTILINESTRING((1 2,3 4 5))",
			err:  ErrNotWKT,
		},
		{
			name: "not a multi linestring",
			s:    "POINT(1 2)",
			err:  ErrIncorrectGeometry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UnmarshalMultiLineString(tc.s)
			if err != tc.err {
				t.Fatalf("incorrect error: %v != %v", err, tc.err)
			}
		})
	}
}

func TestUnmarshalPolygon(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		expected orb.Polygon
	}{
		{
			name:     "empty",
			s:        "POLYGON EMPTY",
			expected: orb.Polygon{},
		},
		{
			name:     "one ring",
			s:        "POLYGON((0 0,1 0,1 1,0 0))",
			expected: orb.Polygon{{{0, 0}, {1, 0}, {1, 1}, {0, 0}}},
		},
		{
			name:     "two rings",
			s:        "POLYGON((1 2,3 4),(5 6,7 8))",
			expected: orb.Polygon{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}},
		},
		{
			name:     "two rings with spaces",
			s:        "POLYGON((1 2,3 4)   ,   (5 6  ,  7 8))",
			expected: orb.Polygon{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// you know it's a polygon
			p, err := UnmarshalPolygon(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			if !p.Equal(tc.expected) {
				t.Log(p)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// lower case
			p, err = UnmarshalPolygon(strings.ToLower(tc.s))
			if err != nil {
				t.Fatal(err)
			}

			if !p.Equal(tc.expected) {
				t.Log(p)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// via generic unmarshal
			geom, err := Unmarshal(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			p = geom.(orb.Polygon)

			if !p.Equal(tc.expected) {
				t.Log(p)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}
		})
	}
}

func TestUnmarshalPolygon_errors(t *testing.T) {
	cases := []struct {
		name string
		s    string
		err  error
	}{
		{
			name: "just name",
			s:    "POLYGON",
			err:  ErrNotWKT,
		},
		{
			name: "too many points",
			s:    "POLYGON((1 2,3 4 5))",
			err:  ErrNotWKT,
		},
		{
			name: "not a polygon",
			s:    "POINT(1 2)",
			err:  ErrIncorrectGeometry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UnmarshalPolygon(tc.s)
			if err != tc.err {
				t.Fatalf("incorrect error: %v != %v", err, tc.err)
			}
		})
	}
}

func TestUnmarshalMutilPolygon(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		expected orb.MultiPolygon
	}{
		{
			name:     "empty",
			s:        "MULTIPOLYGON EMPTY",
			expected: orb.MultiPolygon{},
		},
		{
			name:     "multi-polygon",
			s:        "MULTIPOLYGON(((1 2,3 4)),((5 6,7 8),(1 2,5 4)))",
			expected: orb.MultiPolygon{{{{1, 2}, {3, 4}}}, {{{5, 6}, {7, 8}}, {{1, 2}, {5, 4}}}},
		},
		{
			name:     "multi-polygon with spaces",
			s:        "MULTIPOLYGON(((1 2,3 4))  , 		((5 6,7 8),  (1 2,5 4)))",
			expected: orb.MultiPolygon{{{{1, 2}, {3, 4}}}, {{{5, 6}, {7, 8}}, {{1, 2}, {5, 4}}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// you know it's a multipolygon
			mp, err := UnmarshalMultiPolygon(tc.s)
			if err != nil {
				t.Fatal(err)
			}
			if !mp.Equal(tc.expected) {
				t.Log(mp)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// lower case
			mp, err = UnmarshalMultiPolygon("   " + strings.ToLower(tc.s))
			if err != nil {
				t.Fatal(err)
			}
			if !mp.Equal(tc.expected) {
				t.Log(mp)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// via generic unmarshal
			geom, err := Unmarshal(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			mp = geom.(orb.MultiPolygon)

			if !mp.Equal(tc.expected) {
				t.Log(mp)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}
		})
	}
}

func TestUnmarshalMultiPolygon_errors(t *testing.T) {
	cases := []struct {
		name string
		s    string
		err  error
	}{
		{
			name: "just name",
			s:    "MULTIPOLYGON",
			err:  ErrNotWKT,
		},
		{
			name: "too many points",
			s:    "MULTIPOLYGON(((1 2,3 4 5)))",
			err:  ErrNotWKT,
		},
		{
			name: "missing trailing )",
			s:    "MULTIPOLYGON(((0 1,3 0,4 3,0 4,0 1)), ((3 4,6 3,5 5,3 4)), ((0 0,-1 -2,-3 -2,-2 -1,0 0))",
			err:  ErrNotWKT,
		},
		{
			name: "not a multi polygon",
			s:    "POINT(1 2)",
			err:  ErrIncorrectGeometry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UnmarshalMultiPolygon(tc.s)
			if err != tc.err {
				t.Fatalf("incorrect error: %v != %v", err, tc.err)
			}
		})
	}
}

func TestUnmarshalCollection(t *testing.T) {
	cases := []struct {
		name     string
		s        string
		expected orb.Collection
	}{
		{
			name:     "empty",
			s:        "GEOMETRYCOLLECTION EMPTY",
			expected: orb.Collection{},
		},
		{
			name:     "point and line",
			s:        "GEOMETRYCOLLECTION(POINT(1 2),LINESTRING(3 4,5 6))",
			expected: orb.Collection{orb.Point{1, 2}, orb.LineString{{3, 4}, {5, 6}}},
		},
		{
			name: "lots of things",
			s:    "GEOMETRYCOLLECTION(POINT(1 2),LINESTRING(3 4,5 6),MULTILINESTRING((1 2,3 4),(5 6,7 8)),POLYGON((0 0,1 0,1 1,0 0)),POLYGON((1 2,3 4),(5 6,7 8)),MULTIPOLYGON(((1 2,3 4)),((5 6,7 8),(1 2,5 4))))",
			expected: orb.Collection{
				orb.Point{1, 2},
				orb.LineString{{3, 4}, {5, 6}},
				orb.MultiLineString{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}},
				orb.Polygon{{{0, 0}, {1, 0}, {1, 1}, {0, 0}}},
				orb.Polygon{{{1, 2}, {3, 4}}, {{5, 6}, {7, 8}}},
				orb.MultiPolygon{{{{1, 2}, {3, 4}}}, {{{5, 6}, {7, 8}}, {{1, 2}, {5, 4}}}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// you know it's a collection
			c, err := UnmarshalCollection(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			if !c.Equal(tc.expected) {
				t.Log(c)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// lower case
			c, err = UnmarshalCollection("  " + strings.ToLower(tc.s))
			if err != nil {
				t.Fatal(err)
			}

			if !c.Equal(tc.expected) {
				t.Log(c)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}

			// via generic unmarshal
			geom, err := Unmarshal(tc.s)
			if err != nil {
				t.Fatal(err)
			}

			c = geom.(orb.Collection)

			if !c.Equal(tc.expected) {
				t.Log(c)
				t.Log(tc.expected)
				t.Errorf("incorrect wkt unmarshalling")
			}
		})
	}
}

func TestUnmarshalCollection_errors(t *testing.T) {
	cases := []struct {
		name string
		s    string
		err  error
	}{
		{
			name: "just name",
			s:    "GEOMETRYCOLLECTION",
			err:  ErrNotWKT,
		},
		{
			name: "too many points",
			s:    "GEOMETRYCOLLECTION(POINT(1 2 3))",
			err:  ErrNotWKT,
		},
		{
			name: "missing trailing paren",
			s:    "GEOMETRYCOLLECTION(POINT(1 2 3)",
			err:  ErrNotWKT,
		},
		{
			name: "not a geometry collection",
			s:    "POINT(1 2)",
			err:  ErrIncorrectGeometry,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := UnmarshalCollection(tc.s)
			if err != tc.err {
				t.Fatalf("incorrect error: %v != %v", err, tc.err)
			}
		})
	}
}

func TestSplitOnComma(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "comma",
			input:    "0 1,3 0,4 3,0 4,0 1",
			expected: []string{"0 1", "3 0", "4 3", "0 4", "0 1"},
		},
		{
			name:     "comma spaces",
			input:    "0 1 ,3 0, 4 3 , 0 4  ,   0 1",
			expected: []string{"0 1", "3 0", "4 3", "0 4", "0 1"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var results []string
			err := splitOnComma(tc.input, func(s string) error {
				results = append(results, s)
				return nil
			})
			if err != nil {
				t.Fatalf("impossible error: %v", err)
			}

			if !reflect.DeepEqual(tc.expected, results) {
				t.Log(tc.input)

				data, _ := json.Marshal(results)
				t.Log(string(data))

				t.Log(tc.expected)
				t.Errorf("incorrect results")
			}

		})
	}
}
