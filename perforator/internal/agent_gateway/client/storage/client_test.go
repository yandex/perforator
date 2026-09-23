package storage

import (
	"bytes"
	"fmt"
	"io"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/stretchr/testify/require"

	compressionpb "github.com/yandex/perforator/perforator/proto/lib/compression"
	perforatorstorage "github.com/yandex/perforator/perforator/proto/storage"
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

func TestWithBinaryMetadataCopiesNestedFields(t *testing.T) {
	metadata := &perforatorstorage.BinaryAttributes{Upload: &perforatorstorage.BinaryUploadMetadata{Path: "/lib/libc.so.6"}}
	params := &pushBinaryParams{}
	WithBinaryMetadata(metadata)(params)
	metadata.Upload.Path = "changed"
	require.Equal(t, "/lib/libc.so.6", params.metadata.GetUpload().GetPath())
	WithBinaryMetadata(nil)(params)
	require.Nil(t, params.metadata)
}

func TestCompressionEncoderReuse(t *testing.T) {
	for _, codec := range []string{"zstd", "zstd_1", "zstd_9"} {
		t.Run(codec, func(t *testing.T) {
			conf, err := compressionConfigFromString(codec)
			require.NoError(t, err)
			reference, err := zstd.NewWriter(nil, zstd.WithEncoderLevel(zstd.EncoderLevelFromZstd(conf.zstdLevel)))
			require.NoError(t, err)
			defer reference.Close()
			var reused *zstd.Encoder
			for i := range 3 {
				input := bytes.Repeat([]byte("profile sample values and labels"), 1000+i)
				compressed, err := conf.compressBytes(input)
				require.NoError(t, err)
				encoder := <-conf.zstdPool
				if reused != nil {
					require.Same(t, reused, encoder)
				}
				reused = encoder
				conf.zstdPool <- encoder
				require.Equal(t, reference.EncodeAll(input, nil), compressed)
				reader, err := zstd.NewReader(nil)
				require.NoError(t, err)
				decoded, err := reader.DecodeAll(compressed, nil)
				reader.Close()
				require.NoError(t, err)
				require.Equal(t, input, decoded)
			}
		})
	}
}

func TestCompressionConcurrent(t *testing.T) {
	conf, err := compressionConfigFromString("zstd_3")
	require.NoError(t, err)
	for i := range 8 {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			t.Parallel()
			reader, err := zstd.NewReader(nil)
			require.NoError(t, err)
			defer reader.Close()
			for j := range 5 {
				input := bytes.Repeat([]byte(fmt.Sprintf("profile-%d-%d", i, j)), 1000)
				compressed, err := conf.compressBytes(input)
				require.NoError(t, err)
				decoded, err := reader.DecodeAll(compressed, nil)
				require.NoError(t, err)
				require.Equal(t, input, decoded)
			}
		})
	}
}
