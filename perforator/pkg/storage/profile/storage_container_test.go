package profile

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
	profileproto "github.com/yandex/perforator/perforator/proto/profile"
)

func newTestStorage(t *testing.T) *ProfileStorage {
	t.Helper()
	decompressor, err := zstd.NewReader(nil)
	require.NoError(t, err)
	t.Cleanup(decompressor.Close)
	return &ProfileStorage{
		decompressor: decompressor,
	}
}

func TestUncompressFromContainer(t *testing.T) {
	tests := []struct {
		name        string
		container   *profileproto.ProfileContainer
		expected    []byte
		expectError bool
	}{
		{
			name: "pprof with zstd compression",
			container: &profileproto.ProfileContainer{
				Pprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_Zstd,
					Data:              compressZstd(t, []byte("test profile data")),
				},
			},
			expected:    []byte("test profile data"),
			expectError: false,
		},
		{
			name: "pprof with gzip compression",
			container: &profileproto.ProfileContainer{
				Pprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_Gzip,
					Data:              compressGzip(t, []byte("test profile data")),
				},
			},
			expected:    []byte("test profile data"),
			expectError: false,
		},
		{
			name: "pprof with no compression",
			container: &profileproto.ProfileContainer{
				Pprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_None,
					Data:              []byte("test profile data"),
				},
			},
			expected:    []byte("test profile data"),
			expectError: false,
		},
		{
			name: "yaprof with zstd compression is converted to pprof",
			container: &profileproto.ProfileContainer{
				Yaprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_Zstd,
					Data:              compressZstd(t, []byte("yaprof data")),
				},
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "no payload",
			container: &profileproto.ProfileContainer{
				Pprof:  nil,
				Yaprof: nil,
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "unknown compression method",
			container: &profileproto.ProfileContainer{
				Pprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_Unknown,
					Data:              []byte("test data"),
				},
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newTestStorage(t)
			data, err := storage.uncompressFromContainer(tt.container, "test-profile-id")

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, data)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, data)
			}
		})
	}
}

func TestUncompressPayload(t *testing.T) {
	tests := []struct {
		name        string
		payload     *profileproto.ProfileContainer_Payload
		expected    []byte
		expectError bool
	}{
		{
			name: "zstd compression",
			payload: &profileproto.ProfileContainer_Payload{
				CompressionMethod: compressionpb.CompressionMethod_Zstd,
				Data:              compressZstd(t, []byte("test data")),
			},
			expected:    []byte("test data"),
			expectError: false,
		},
		{
			name: "gzip compression",
			payload: &profileproto.ProfileContainer_Payload{
				CompressionMethod: compressionpb.CompressionMethod_Gzip,
				Data:              compressGzip(t, []byte("test data")),
			},
			expected:    []byte("test data"),
			expectError: false,
		},
		{
			name: "no compression",
			payload: &profileproto.ProfileContainer_Payload{
				CompressionMethod: compressionpb.CompressionMethod_None,
				Data:              []byte("test data"),
			},
			expected:    []byte("test data"),
			expectError: false,
		},
		{
			name: "unknown compression",
			payload: &profileproto.ProfileContainer_Payload{
				CompressionMethod: compressionpb.CompressionMethod_Unknown,
				Data:              []byte("test data"),
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newTestStorage(t)
			data, err := storage.uncompressPayload(tt.payload, "test-id")

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, data)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, data)
			}
		})
	}
}

func TestUncompressZstd(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expected    []byte
		expectError bool
	}{
		{
			name:        "valid zstd data",
			data:        compressZstd(t, []byte("test data")),
			expected:    []byte("test data"),
			expectError: false,
		},
		{
			name:        "invalid zstd data",
			data:        []byte("not zstd data"),
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newTestStorage(t)
			data, err := storage.uncompressZstd(tt.data)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, data)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, data)
			}
		})
	}
}

func TestUncompressGzip(t *testing.T) {
	tests := []struct {
		name        string
		data        []byte
		expected    []byte
		expectError bool
	}{
		{
			name:        "valid gzip data",
			data:        compressGzip(t, []byte("test data")),
			expected:    []byte("test data"),
			expectError: false,
		},
		{
			name:        "invalid gzip data",
			data:        []byte("not gzip data"),
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newTestStorage(t)
			data, err := storage.uncompressGzip(tt.data)

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, data)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.expected, data)
			}
		})
	}
}

func compressZstd(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	encoder, err := zstd.NewWriter(&buf)
	require.NoError(t, err)
	_, err = encoder.Write(data)
	require.NoError(t, err)
	require.NoError(t, encoder.Close())
	return buf.Bytes()
}

func compressGzip(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)
	_, err := writer.Write(data)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return buf.Bytes()
}
