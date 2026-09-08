package process

import (
	"testing"

	"google.golang.org/protobuf/proto"

	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
)

func TestGetBinaryAttributes(t *testing.T) {
	const host = "upload-host.example"

	for _, tc := range []struct {
		name     string
		path     string
		wantPath string
		filename string
	}{
		{name: "shared library", path: "/usr/lib/libc.so.6", wantPath: "/usr/lib/libc.so.6", filename: "libc.so.6"},
		{name: "executable", path: "/app/bin/server", wantPath: "/app/bin/server", filename: "server"},
		{name: "vdso", path: "[vdso]", wantPath: "[vdso]", filename: "[vdso]"},
		{name: "deleted", path: "/usr/lib/libc.so.6 (deleted)", wantPath: "/usr/lib/libc.so.6", filename: "libc.so.6"},
		{name: "internal marker", path: "/app (deleted)/lib.so", wantPath: "/app (deleted)/lib.so", filename: "lib.so"},
		{name: "one suffix", path: "/lib.so (deleted) (deleted)", wantPath: "/lib.so (deleted)", filename: "lib.so (deleted)"},
		{name: "empty"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			expected := &perforatorstorage.BinaryAttributes{Upload: &perforatorstorage.BinaryUploadMetadata{Host: host}}
			if tc.wantPath != "" {
				expected.Upload.Path = tc.wantPath
				expected.Upload.Filename = tc.filename
			}
			if got := getBinaryAttributes(tc.path, host); !proto.Equal(expected, got) {
				t.Fatalf("attributes = %v, want %v", got, expected)
			}
		})
	}
}

func TestGetBinaryAttributesWithoutHost(t *testing.T) {
	expected := &perforatorstorage.BinaryAttributes{Upload: &perforatorstorage.BinaryUploadMetadata{Path: "/usr/lib/libc.so.6", Filename: "libc.so.6"}}
	if got := getBinaryAttributes("/usr/lib/libc.so.6", ""); !proto.Equal(expected, got) {
		t.Fatalf("attributes = %v, want %v", got, expected)
	}
}
