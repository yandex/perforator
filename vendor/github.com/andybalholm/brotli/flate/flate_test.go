package flate

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/andybalholm/brotli/matchfinder"
)

func test(t *testing.T, filename string, m matchfinder.MatchFinder, blockSize int) {
	data, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	b := new(bytes.Buffer)
	w := &matchfinder.Writer{
		Dest:        b,
		MatchFinder: m,
		Encoder:     NewEncoder(),
		BlockSize:   blockSize,
	}
	w.Write(data)
	w.Close()
	compressed := b.Bytes()
	sr := flate.NewReader(bytes.NewReader(compressed))
	decompressed, err := io.ReadAll(sr)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decompressed, data) {
		t.Fatal("decompressed output doesn't match")
	}
}

func TestEncodeHuffmanOnly(t *testing.T) {
	test(t, "../testdata/Isaac.Newton-Opticks.txt", matchfinder.NoMatchFinder{}, 1<<16)
}

func TestWriterLevels(t *testing.T) {
	data, err := os.ReadFile("../testdata/Isaac.Newton-Opticks.txt")
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i < 10; i++ {
		b := new(bytes.Buffer)
		w := NewWriter(b, i)
		w.Write(data)
		w.Close()
		compressed := b.Bytes()
		sr := flate.NewReader(bytes.NewReader(compressed))
		decompressed, err := io.ReadAll(sr)
		if err != nil {
			t.Fatalf("error decompressing level %d: %v", i, err)
		}
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("decompressed output doesn't match on level %d", i)
		}
	}
}

func TestGZIPWriterLevels(t *testing.T) {
	data, err := os.ReadFile("../testdata/Isaac.Newton-Opticks.txt")
	if err != nil {
		t.Fatal(err)
	}

	for i := 1; i < 10; i++ {
		b := new(bytes.Buffer)
		w := NewGZIPWriter(b, i)
		w.Write(data)
		w.Close()
		compressed := b.Bytes()
		sr, err := gzip.NewReader(bytes.NewReader(compressed))
		if err != nil {
			t.Fatalf("error creating gzip reader: %v", err)
		}
		decompressed, err := io.ReadAll(sr)
		if err != nil {
			t.Fatalf("error decompressing level %d: %v", i, err)
		}
		if !bytes.Equal(decompressed, data) {
			t.Fatalf("decompressed output doesn't match on level %d", i)
		}
	}
}

func TestLongMatches(t *testing.T) {
	// DEFLATE can't encode a match longer than 258 bytes.
	inputs := [][]byte{
		make([]byte, 259),
		make([]byte, 260),
		make([]byte, 1<<17),
		bytes.Repeat([]byte("the quick brown fox jumps over the lazy dog. "), 8),
	}

	for _, data := range inputs {
		for i := 1; i < 10; i++ {
			b := new(bytes.Buffer)
			w := NewWriter(b, i)
			w.Write(data)
			w.Close()
			decompressed, err := io.ReadAll(flate.NewReader(b))
			if err != nil {
				t.Fatalf("error decompressing %d bytes on level %d: %v", len(data), i, err)
			}
			if !bytes.Equal(decompressed, data) {
				t.Fatalf("decompressed output doesn't match for %d bytes on level %d", len(data), i)
			}

			b.Reset()
			w = NewGZIPWriter(b, i)
			w.Write(data)
			w.Close()
			sr, err := gzip.NewReader(b)
			if err != nil {
				t.Fatalf("error creating gzip reader: %v", err)
			}
			decompressed, err = io.ReadAll(sr)
			if err != nil {
				t.Fatalf("error decompressing gzip, %d bytes on level %d: %v", len(data), i, err)
			}
			if !bytes.Equal(decompressed, data) {
				t.Fatalf("decompressed gzip output doesn't match for %d bytes on level %d", len(data), i)
			}
		}
	}
}

func benchmark(b *testing.B, filename string, m matchfinder.MatchFinder, blockSize int) {
	b.StopTimer()
	b.ReportAllocs()
	data, err := os.ReadFile(filename)
	if err != nil {
		b.Fatal(err)
	}

	b.SetBytes(int64(len(data)))
	buf := new(bytes.Buffer)
	w := &matchfinder.Writer{
		Dest:        buf,
		MatchFinder: m,
		Encoder:     NewEncoder(),
		BlockSize:   blockSize,
	}
	w.Write(data)
	w.Close()
	b.ReportMetric(float64(len(data))/float64(buf.Len()), "ratio")
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		w.Reset(io.Discard)
		w.Write(data)
		w.Close()
	}
}

func BenchmarkEncodeHuffmanOnly(b *testing.B) {
	benchmark(b, "../testdata/Isaac.Newton-Opticks.txt", matchfinder.NoMatchFinder{}, 1<<20)
}

func BenchmarkWriterLevels(b *testing.B) {
	opticks, err := os.ReadFile("../testdata/Isaac.Newton-Opticks.txt")
	if err != nil {
		b.Fatal(err)
	}

	for level := 1; level <= 9; level++ {
		buf := new(bytes.Buffer)
		w := NewWriter(buf, level)
		w.Write(opticks)
		w.Close()
		b.Run(fmt.Sprintf("%d", level), func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(len(opticks))/float64(buf.Len()), "ratio")
			b.SetBytes(int64(len(opticks)))
			for i := 0; i < b.N; i++ {
				w.Reset(io.Discard)
				w.Write(opticks)
				w.Close()
			}
		})
	}
}

func BenchmarkStdlibLevels(b *testing.B) {
	opticks, err := os.ReadFile("../testdata/Isaac.Newton-Opticks.txt")
	if err != nil {
		b.Fatal(err)
	}

	for level := 1; level <= 9; level++ {
		buf := new(bytes.Buffer)
		w, _ := flate.NewWriter(buf, level)
		w.Write(opticks)
		w.Close()
		b.Run(fmt.Sprintf("%d", level), func(b *testing.B) {
			b.ReportAllocs()
			b.ReportMetric(float64(len(opticks))/float64(buf.Len()), "ratio")
			b.SetBytes(int64(len(opticks)))
			for i := 0; i < b.N; i++ {
				w.Reset(io.Discard)
				w.Write(opticks)
				w.Close()
			}
		})
	}
}
