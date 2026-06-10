package storage

import (
	"bytes"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
)

func TestCompressionConfigFromString(t *testing.T) {
	for _, test := range []struct {
		name      string
		input     string
		enabled   bool
		codec     compressionpb.CompressionMethod
		codecName string
		zstdLevel int
	}{
		{
			name:  "disabled",
			input: "",
		},
		{
			name:      "zstd default",
			input:     "zstd",
			enabled:   true,
			codec:     compressionpb.CompressionMethod_Zstd,
			codecName: "zstd",
			zstdLevel: 6,
		},
		{
			name:      "zstd explicit level",
			input:     "zstd_3",
			enabled:   true,
			codec:     compressionpb.CompressionMethod_Zstd,
			codecName: "zstd_3",
			zstdLevel: 3,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			conf, err := compressionConfigFromString(test.input)
			require.NoError(t, err)
			if !test.enabled {
				require.Nil(t, conf)
				return
			}

			require.NotNil(t, conf)
			require.Equal(t, test.codec, conf.codec)
			require.Equal(t, test.codecName, conf.codecName)
			require.Equal(t, test.zstdLevel, conf.zstdLevel)
		})
	}
}

func TestCompressionConfigFromStringInvalidCodec(t *testing.T) {
	conf, err := compressionConfigFromString("gzip")
	require.ErrorContains(t, err, "unrecognized compression codec gzip")
	require.Nil(t, conf)
}

func TestCompressionConfigCompressBytes(t *testing.T) {
	conf, err := compressionConfigFromString("zstd_3")
	require.NoError(t, err)
	require.NotNil(t, conf)

	original := []byte("hello compressed profile")
	compressed, err := conf.compressBytes(original)
	require.NoError(t, err)
	require.NotEqual(t, original, compressed)

	reader, err := zstd.NewReader(bytes.NewReader(compressed))
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, original, decompressed)
}

func TestCompressionConfigZstdWriter(t *testing.T) {
	conf, err := compressionConfigFromString("zstd_3")
	require.NoError(t, err)
	require.NotNil(t, conf)

	var compressed bytes.Buffer
	writer, err := conf.newWriter(&compressed)
	require.NoError(t, err)

	_, err = writer.Write([]byte("hello compressed binary"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	reader, err := zstd.NewReader(&compressed)
	require.NoError(t, err)
	defer reader.Close()

	decompressed, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, []byte("hello compressed binary"), decompressed)
}
