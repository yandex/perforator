package wkt

import (
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/paulmach/orb"
)

func BenchmarkUnmarshalPoint(b *testing.B) {
	var mp orb.MultiPolygon
	loadJSON(b, "testdata/polygon.json", &mp)

	text := MarshalString(orb.Point{-81.60644531, 41.51377887})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Unmarshal(text)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkUnmarshalLineString_small(b *testing.B) {
	ls := orb.LineString{{1, 2}, {3, 4}}

	text := MarshalString(ls)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Unmarshal(text)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkUnmarshalLineString(b *testing.B) {
	var mp orb.MultiPolygon
	loadJSON(b, "testdata/polygon.json", &mp)

	text := MarshalString(orb.LineString(mp[0][0]))

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Unmarshal(text)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkUnmarshalPolygon(b *testing.B) {
	var mp orb.MultiPolygon
	loadJSON(b, "testdata/polygon.json", &mp)

	text := MarshalString(mp[0])

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Unmarshal(text)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkUnmarshalMultiPolygon_small(b *testing.B) {
	mp := orb.MultiPolygon{{{{1, 2}, {3, 4}}}, {{{5, 6}, {7, 8}}, {{1, 2}, {5, 4}}}}

	text := MarshalString(mp)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Unmarshal(text)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func BenchmarkUnmarshalMultiPolygon(b *testing.B) {
	var mp orb.MultiPolygon
	loadJSON(b, "testdata/polygon.json", &mp)

	text := MarshalString(mp)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Unmarshal(text)
		if err != nil {
			b.Fatalf("unexpected error: %v", err)
		}
	}
}

func loadJSON(tb testing.TB, filename string, obj interface{}) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		tb.Fatalf("failed to load mvt file: %v", err)
	}

	err = json.Unmarshal(data, obj)
	if err != nil {
		tb.Fatalf("unmarshal error: %v", err)
	}
}
