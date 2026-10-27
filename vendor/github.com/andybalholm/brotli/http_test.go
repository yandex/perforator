package brotli

import (
	"net/http"
	"testing"
)

var acceptEncodingCases = []struct {
	acceptEncoding string
	want           string
}{
	{"br;q=0.5, gzip", "gzip"},
	{"br;q=0.5, *", "gzip"},
	{"br;q=0, gzip", "gzip"},
	{"br;q=0, *", "gzip"},
	{"br", "br"},
	{"BR", "br"},
	{"gzip", "gzip"},
	{"gzip,br", "br"},
	{"gzip,BR", "br"},
	{"*;q=0.5, br;q=0", "gzip"},
	{"*, br;q=0", "gzip"},
	{"*;q=0.5, br", "br"},
	{"*;q=0.1, gzip;q=0.5", "gzip"},
	{"*, br;q=0, gzip;q=0", ""},
}

func TestNegotiateContentEncoding(t *testing.T) {
	offers := []string{"br", "gzip"}
	for _, c := range acceptEncodingCases {
		r, _ := http.NewRequest("GET", "http://example.com", nil)
		r.Header.Set("Accept-Encoding", c.acceptEncoding)
		got := negotiateContentEncoding(r, offers)
		if got != c.want {
			t.Errorf("acceptEncoding=%q, got=%q, want=%q", c.acceptEncoding, got, c.want)
		}
	}
}
