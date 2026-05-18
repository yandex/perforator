package profile

import (
	"bytes"
	"compress/gzip"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

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

func TestBundleFromContainer(t *testing.T) {
	tests := []struct {
		name           string
		container      *profileproto.ProfileContainer
		expectedPprof  []byte
		expectedYaprof []byte
		expectError    bool
	}{
		{
			name: "pprof with zstd compression",
			container: &profileproto.ProfileContainer{
				Pprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_Zstd,
					Data:              compressZstd(t, []byte("test profile data")),
				},
			},
			expectedPprof: []byte("test profile data"),
			expectError:   false,
		},
		{
			name: "pprof with gzip compression",
			container: &profileproto.ProfileContainer{
				Pprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_Gzip,
					Data:              compressGzip(t, []byte("test profile data")),
				},
			},
			expectedPprof: []byte("test profile data"),
			expectError:   false,
		},
		{
			name: "pprof with no compression",
			container: &profileproto.ProfileContainer{
				Pprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_None,
					Data:              []byte("test profile data"),
				},
			},
			expectedPprof: []byte("test profile data"),
			expectError:   false,
		},
		{
			name: "yaprof only returns bundle with yaprof",
			container: &profileproto.ProfileContainer{
				Yaprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_Zstd,
					Data:              compressZstd(t, []byte("yaprof data")),
				},
			},
			expectedYaprof: []byte("yaprof data"),
			expectError:    false,
		},
		{
			name: "both pprof and yaprof",
			container: &profileproto.ProfileContainer{
				Pprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_None,
					Data:              []byte("pprof data"),
				},
				Yaprof: &profileproto.ProfileContainer_Payload{
					CompressionMethod: compressionpb.CompressionMethod_None,
					Data:              []byte("yaprof data"),
				},
			},
			expectedPprof:  []byte("pprof data"),
			expectedYaprof: []byte("yaprof data"),
			expectError:    false,
		},
		{
			name: "no payload",
			container: &profileproto.ProfileContainer{
				Pprof:  nil,
				Yaprof: nil,
			},
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
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newTestStorage(t)
			bundle, err := storage.bundleFromContainer(tt.container, "test-profile-id")

			if tt.expectError {
				require.Error(t, err)
				require.Nil(t, bundle)
			} else {
				require.NoError(t, err)
				require.NotNil(t, bundle)
				if tt.expectedPprof != nil {
					require.Equal(t, tt.expectedPprof, bundle.GetPprof())
				}
				if tt.expectedYaprof != nil {
					require.Equal(t, tt.expectedYaprof, bundle.GetYaprof())
				}
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

func TestDetectCompressionMethod(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected compressionpb.CompressionMethod
	}{
		{
			name:     "zstd data",
			data:     compressZstd(t, []byte("test data")),
			expected: compressionpb.CompressionMethod_Zstd,
		},
		{
			name:     "gzip data",
			data:     compressGzip(t, []byte("test data")),
			expected: compressionpb.CompressionMethod_Gzip,
		},
		{
			name:     "uncompressed data",
			data:     []byte("test data"),
			expected: compressionpb.CompressionMethod_None,
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: compressionpb.CompressionMethod_None,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectCompressionMethod(tt.data)
			require.Equal(t, tt.expected, result)
		})
	}
}

func TestWrapInContainer(t *testing.T) {
	tests := []struct {
		name         string
		pprofBody    []byte
		yaprofBody   []byte
		expectPprof  bool
		expectYaprof bool
	}{
		{
			name:        "uncompressed pprof body",
			pprofBody:   []byte("test profile data"),
			expectPprof: true,
		},
		{
			name:        "zstd compressed pprof body",
			pprofBody:   compressZstd(t, []byte("test profile data")),
			expectPprof: true,
		},
		{
			name:        "gzip compressed pprof body",
			pprofBody:   compressGzip(t, []byte("test profile data")),
			expectPprof: true,
		},
		{
			name:         "yaprof body only",
			yaprofBody:   []byte("yaprof data"),
			expectYaprof: true,
		},
		{
			name:         "both pprof and yaprof",
			pprofBody:    []byte("pprof data"),
			yaprofBody:   []byte("yaprof data"),
			expectPprof:  true,
			expectYaprof: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newTestStorage(t)

			containerBytes, err := storage.wrapInContainer(tt.pprofBody, tt.yaprofBody)
			require.NoError(t, err)
			require.NotEmpty(t, containerBytes)

			container := &profileproto.ProfileContainer{}
			err = proto.Unmarshal(containerBytes, container)
			require.NoError(t, err)

			if tt.expectPprof {
				require.NotNil(t, container.Pprof)
				require.Equal(t, tt.pprofBody, container.Pprof.Data)
				require.Equal(t, detectCompressionMethod(tt.pprofBody), container.Pprof.CompressionMethod)
			} else {
				require.Nil(t, container.Pprof)
			}

			if tt.expectYaprof {
				require.NotNil(t, container.Yaprof)
				require.Equal(t, tt.yaprofBody, container.Yaprof.Data)
				require.Equal(t, detectCompressionMethod(tt.yaprofBody), container.Yaprof.CompressionMethod)
			} else {
				require.Nil(t, container.Yaprof)
			}
		})
	}
}

func TestWrapInContainerRoundTrip(t *testing.T) {
	originalData := []byte("original profile data for round trip test")

	tests := []struct {
		name string
		body []byte
	}{
		{
			name: "uncompressed round trip",
			body: originalData,
		},
		{
			name: "zstd compressed round trip",
			body: compressZstd(t, originalData),
		},
		{
			name: "gzip compressed round trip",
			body: compressGzip(t, originalData),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newTestStorage(t)

			containerBytes, err := storage.wrapInContainer(tt.body, nil)
			require.NoError(t, err)

			container := &profileproto.ProfileContainer{}
			err = proto.Unmarshal(containerBytes, container)
			require.NoError(t, err)
			require.NotNil(t, container.Pprof)

			bundle, err := storage.bundleFromContainer(container, "test-round-trip-id")
			require.NoError(t, err)
			require.Equal(t, originalData, bundle.GetPprof())
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
