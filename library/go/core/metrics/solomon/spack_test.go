package solomon

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpackStartTimeObjectLayout(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("64-bit layout check")
	}
	require.Equal(t, uintptr(48), unsafe.Sizeof(baseMetric{}))
	require.Equal(t, uintptr(40), unsafe.Sizeof(MetricsOpts{}))
}

func TestSpackVersion13(t *testing.T) {
	counter := NewCounter("requests", 42, WithRated(true), WithMemOnly(),
		WithTimestamp(time.Unix(0x11223344, 0)),
		WithTags(map[string]string{"host": "a\x00b", "sensor": "r\x00q"}))
	metrics := NewMetrics([]Metric{counter.Snapshot()},
		WithTimestamp(time.Unix(0x55667788, 0)),
		WithCommonLabels(map[string]string{"host": "a\x00b"}))

	// Layout from monlib/encode/spack/spack_v1_{encoder,decoder}.cpp:
	// pool counts in the header and length-delimited strings.
	expected := []byte{
		0x53, 0x50, 0x03, 0x01, 0x18, 0x00, 0x00, 0x00,
		0x02, 0x00, 0x00, 0x00, // name count
		0x02, 0x00, 0x00, 0x00, // value count
		0x01, 0x00, 0x00, 0x00, // metric count
		0x01, 0x00, 0x00, 0x00, // point count
		4, 'h', 'o', 's', 't', 6, 's', 'e', 'n', 's', 'o', 'r',
		3, 'a', 0, 'b', 3, 'r', 0, 'q',
		0x88, 0x77, 0x66, 0x55, // common time
		1, 0, 0, // common label: host=a\0b
		0x0e, 0x01, // RATE with timestamp, mem-only
		1, 1, 1, // metric label: sensor=r\0q (host inherited)
		0x44, 0x33, 0x22, 0x11, // timestamp
		42, 0, 0, 0, 0, 0, 0, 0, // value
	}
	for _, compression := range []CompressionType{CompressionNone, CompressionLz4} {
		t.Run(fmt.Sprint(compression), func(t *testing.T) {
			want := bytes.Clone(expected)
			want[7] = byte(compression)
			if compression == CompressionLz4 {
				want = append(want[:HeaderSize], compress(t, uint8(compression), string(expected[HeaderSize:]))...)
			}
			var buf bytes.Buffer
			n, err := NewSpackEncoder(context.Background(), compression, &metrics, WithVersion13()).Encode(&buf)
			require.NoError(t, err)
			require.Equal(t, want, buf.Bytes())
			require.Equal(t, len(want), n)
		})
	}
}

func TestSpackVersion14(t *testing.T) {
	const commonStartSeconds uint32 = 0x11223344
	const metricStartSeconds uint32 = 0x55667788
	shared := NewCounter("shared", 10, WithRated(true), WithStartTime(commonStartSeconds))
	gauge := NewGauge("gauge", 1, WithStartTime(metricStartSeconds))
	own := NewCounter("own", 20, WithRated(true), WithStartTime(metricStartSeconds))
	withoutStart := NewCounter("none", 30, WithRated(true), WithStartTime(0))
	metrics := NewMetrics([]Metric{&shared, &gauge, own.Snapshot(), &withoutStart},
		WithCommonStartTime(time.Unix(int64(commonStartSeconds), 0)))

	expected := []byte{
		0x53, 0x50, 0x04, 0x01, 0x18, 0x00, 0x00, 0x00,
		0x01, 0x00, 0x00, 0x00, // name count
		0x04, 0x00, 0x00, 0x00, // value count
		0x04, 0x00, 0x00, 0x00, // metric count
		0x04, 0x00, 0x00, 0x00, // point count
		6, 's', 'e', 'n', 's', 'o', 'r',
		6, 's', 'h', 'a', 'r', 'e', 'd',
		5, 'g', 'a', 'u', 'g', 'e',
		3, 'o', 'w', 'n',
		4, 'n', 'o', 'n', 'e',
		0, 0, 0, 0, // common time
		0x44, 0x33, 0x22, 0x11, // common start time
		0,                // common labels
		0x0d, 0, 1, 0, 0, // RATE with the common start time
		10, 0, 0, 0, 0, 0, 0, 0,
		0x05, 0, 1, 0, 1, // GAUGE ignores its start time
		0, 0, 0, 0, 0, 0, 0xf0, 0x3f,
		0x0d, 2, 0x88, 0x77, 0x66, 0x55, 1, 0, 2, // RATE with its own start time
		20, 0, 0, 0, 0, 0, 0, 0,
		0x0d, 0, 1, 0, 3, // RATE without a start time
		30, 0, 0, 0, 0, 0, 0, 0,
	}
	for _, compression := range []CompressionType{CompressionNone, CompressionLz4} {
		t.Run(fmt.Sprint(compression), func(t *testing.T) {
			want := bytes.Clone(expected)
			want[7] = byte(compression)
			if compression == CompressionLz4 {
				want = append(want[:HeaderSize], compress(t, uint8(compression), string(expected[HeaderSize:]))...)
			}
			var buf bytes.Buffer
			n, err := NewSpackEncoder(context.Background(), compression, &metrics, WithVersion14()).Encode(&buf)
			require.NoError(t, err)
			require.Equal(t, want, buf.Bytes())
			require.Equal(t, len(want), n)
		})
	}
}

func TestSpackVersion14ZeroCommonStartTimeUsesDefault(t *testing.T) {
	metrics := NewMetrics(nil, WithCommonStartTime(time.Unix(0, 0)))
	var buf bytes.Buffer
	_, err := NewSpackEncoder(context.Background(), CompressionNone, &metrics, WithVersion14()).Encode(&buf)
	require.NoError(t, err)
	require.Equal(t, packageInitTimeSeconds, binary.LittleEndian.Uint32(buf.Bytes()[HeaderSize+4:]))
}

func readSpackStringPool(t *testing.T, r *bytes.Reader, count uint32) []string {
	t.Helper()
	pool := make([]string, count)
	for i := range pool {
		n, err := binary.ReadUvarint(r)
		require.NoError(t, err)
		require.LessOrEqual(t, n, uint64(r.Len()))
		value := make([]byte, int(n))
		_, err = io.ReadFull(r, value)
		require.NoError(t, err)
		pool[i] = string(value)
	}
	return pool
}

func TestSpackVersion13StringPools(t *testing.T) {
	for _, label := range []string{"", "\x00", "a\x00b", "метрика", strings.Repeat("x", 127),
		strings.Repeat("x", 128), strings.Repeat("x", 16383), strings.Repeat("x", 16384)} {
		t.Run(fmt.Sprintf("%d/%q", len(label), label[:min(len(label), 8)]), func(t *testing.T) {
			g := NewGauge("ignored", 1, WithUseNameTag(), WithTags(map[string]string{"name": label, label: label}))
			metrics := NewMetrics([]Metric{&g}, WithCommonLabels(map[string]string{label: label}))
			var buf bytes.Buffer
			_, err := NewSpackEncoder(context.Background(), CompressionNone, &metrics, WithVersion13()).Encode(&buf)
			require.NoError(t, err)
			data := buf.Bytes()
			require.Equal(t, uint32(2), binary.LittleEndian.Uint32(data[8:12]))
			require.Equal(t, uint32(1), binary.LittleEndian.Uint32(data[12:16]))
			r := bytes.NewReader(data[HeaderSize:])
			require.Equal(t, []string{label, "name"}, readSpackStringPool(t, r, 2))
			require.Equal(t, []string{label}, readSpackStringPool(t, r, 1))
			rest, err := io.ReadAll(r)
			require.NoError(t, err)
			require.Equal(t, []byte{
				0, 0, 0, 0, // common time
				1, 0, 0, // common label
				5, 0, 1, 1, 0, // GAUGE, flags, name=label
				0, 0, 0, 0, 0, 0, 0xf0, 0x3f, // 1.0
			}, rest)
		})
	}
}

func TestSpackVersion13SeriesAndPoolIndexes(t *testing.T) {
	const count = 130 // Pool indexes cross the one-byte ULEB128 boundary.
	list := make([]Metric, count)
	values := make([]string, count)
	for i := range list {
		values[i] = fmt.Sprintf("m%d", i)
		s := NewCounterSeries(values[i], []CounterPoint{
			{Timestamp: time.Unix(2000, 0), Value: 42},
			{Timestamp: time.Unix(2001, 0), Value: 43},
		}, WithRated(true))
		list[i] = &s
	}
	metrics := NewMetrics(list)
	var buf bytes.Buffer
	_, err := NewSpackEncoder(context.Background(), CompressionNone, &metrics, WithVersion13()).Encode(&buf)
	require.NoError(t, err)
	data := buf.Bytes()
	require.Equal(t, uint32(1), binary.LittleEndian.Uint32(data[8:12]))
	require.Equal(t, uint32(count), binary.LittleEndian.Uint32(data[12:16]))
	require.Equal(t, uint32(count), binary.LittleEndian.Uint32(data[16:20]))
	require.Equal(t, uint32(2*count), binary.LittleEndian.Uint32(data[20:24]))
	r := bytes.NewReader(data[HeaderSize:])
	require.Equal(t, []string{"sensor"}, readSpackStringPool(t, r, 1))
	require.Equal(t, values, readSpackStringPool(t, r, count))
	common := make([]byte, 5)
	_, err = io.ReadFull(r, common)
	require.NoError(t, err)
	require.Equal(t, make([]byte, 5), common)
	for i := range list {
		prefix := make([]byte, 4)
		_, err = io.ReadFull(r, prefix)
		require.NoError(t, err)
		require.Equal(t, []byte{0x0f, 0, 1, 0}, prefix) // RATE series, flags, label count and name index
		index, err := binary.ReadUvarint(r)
		require.NoError(t, err)
		require.Equal(t, uint64(i), index)
		points := make([]byte, 25)
		_, err = io.ReadFull(r, points)
		require.NoError(t, err)
		require.Equal(t, []byte{
			2,
			0xd0, 7, 0, 0, 42, 0, 0, 0, 0, 0, 0, 0,
			0xd1, 7, 0, 0, 43, 0, 0, 0, 0, 0, 0, 0,
		}, points)
	}
	require.Zero(t, r.Len())
}

func TestSpackVersion13Empty(t *testing.T) {
	for _, metrics := range []*Metrics{nil, {}} {
		var buf bytes.Buffer
		n, err := NewSpackEncoder(context.Background(), CompressionNone, metrics, WithVersion13()).Encode(&buf)
		require.NoError(t, err)
		require.Equal(t, 29, n) // header, common time, label count
		expected := make([]byte, 29)
		copy(expected, []byte{0x53, 0x50, 0x03, 0x01, 24, 0})
		require.Equal(t, expected, buf.Bytes())
	}
}

func Test_metrics_encode(t *testing.T) {
	expectHeader := []byte{
		0x53, 0x50, // magic
		0x01, 0x01, // version
		0x18, 0x00, // header size
		0x0,                // time precision
		0x0,                // compression algorithm
		0x7, 0x0, 0x0, 0x0, // label names size
		0x8, 0x0, 0x0, 0x0, // label values size
		0x1, 0x0, 0x0, 0x0, // metric count
		0x1, 0x0, 0x0, 0x0, // point count
		// label names pool
		0x73, 0x65, 0x6e, 0x73, 0x6f, 0x72, 0x0, // "sensor"
		// label values pool
		0x6d, 0x79, 0x67, 0x61, 0x75, 0x67, 0x65, 0x0, // "gauge"
	}

	type commonLabels struct {
		count  byte
		labels [][]byte
	}

	testCases := []struct {
		name               string
		metrics            *Metrics
		version            spackVersion
		expectHeader       []byte
		expectCommonTime   []byte
		expectCommonLabels commonLabels
		expectMetrics      [][]byte
		expectWritten      int
	}{
		{
			"common-ts+gauge",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewGauge("mygauge", 43)
						return &g
					}(),
				},
				timestamp: timeAsRef(time.Unix(1500000000, 0)),
			},
			version11,
			expectHeader,
			[]byte{0x0, 0x2f, 0x68, 0x59}, // common time  /1500000000
			commonLabels{
				0x0,
				[][]byte{},
			},
			[][]byte{
				{
					0x5, // types
					0x0, // flags
					0x1, // labels index size
					0x0, // indexes of name labels
					0x0, // indexes of value labels

					0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x45, 0x40, // 43  // metrics value

				},
			},
			57,
		},
		{
			"gauge+ts",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewGauge("mygauge", 43, WithTimestamp(time.Unix(1657710476, 0)))
						return &g
					}(),
				},
			},
			version11,
			expectHeader,
			[]byte{0x0, 0x0, 0x0, 0x0}, // common time
			commonLabels{
				0x0,
				[][]byte{},
			},
			[][]byte{
				{
					0x6, // uint8(typeGauge << 2) | uint8(valueTypeOneWithTS)
					0x0, // flags
					0x1, // labels index size
					0x0, // indexes of name labels
					0x0, // indexes of value labels

					0x8c, 0xa7, 0xce, 0x62, // metric ts
					0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x45, 0x40, // 43  // metrics value

				},
			},
			61,
		},
		{
			"common-ts+gauge+ts",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewGauge("mygauge", 43, WithTimestamp(time.Unix(1657710476, 0)))
						return &g
					}(),
					func() Metric {
						g := NewGauge("mygauge", 42, WithTimestamp(time.Unix(1500000000, 0)))
						return &g
					}(),
				},
				timestamp: timeAsRef(time.Unix(1500000000, 0)),
			},
			version11,
			expectHeader,
			[]byte{0x0, 0x2f, 0x68, 0x59}, // common time  /1500000000
			commonLabels{
				0x0,
				[][]byte{},
			},
			[][]byte{
				{
					0x6, // types
					0x0, // flags
					0x1, // labels index size
					0x0, // indexes of name labels
					0x0, // indexes of value labels

					0x8c, 0xa7, 0xce, 0x62, // metric ts
					0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x45, 0x40, // 43  // metrics value

				},
				{
					0x6, // types
					0x0, // flags
					0x1, // labels index size
					0x0, // indexes of name labels
					0x0, // indexes of value labels

					0x0, 0x2f, 0x68, 0x59, // metric ts
					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x45, 0x40, //42 // metrics value

				},
			},
			78,
		},
		{
			"gauge+memOnly",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewGauge("mygauge", 43, WithTimestamp(time.Unix(1657710476, 0)), WithMemOnly())
						return &g
					}(),
				},
			},
			version11,
			expectHeader,
			[]byte{0x0, 0x0, 0x0, 0x0}, // common time
			commonLabels{
				0x0,
				[][]byte{},
			},
			[][]byte{
				{
					0x6, // uint8(typeGauge << 2) | uint8(valueTypeOneWithTS)
					0x1, // flags
					0x1, // labels index size
					0x0, // indexes of name labels
					0x0, // indexes of value labels

					0x8c, 0xa7, 0xce, 0x62, // metric ts
					0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x45, 0x40, // 43  // metrics value

				},
			},
			61,
		},
		{
			"gauge+commonLabels",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewGauge("mygauge", 43)
						return &g
					}(),
				},
				commonLabels: map[string]string{
					"project": "solomon",
				},
			},
			version11,
			[]byte{
				0x53, 0x50, // magic
				0x01, 0x01, // version
				0x18, 0x00, // header size
				0x0,                // time precision
				0x0,                // compression algorithm
				0xf, 0x0, 0x0, 0x0, // label names size (15 bytes: "project\0sensor\0")
				0x10, 0x0, 0x0, 0x0, // label values size (16 bytes: "solomon\0mygauge\0")
				0x1, 0x0, 0x0, 0x0, // metric count
				0x1, 0x0, 0x0, 0x0, // point count
				// label names pool
				0x70, 0x72, 0x6f, 0x6a, 0x65, 0x63, 0x74, 0x0, // "project\0"
				0x73, 0x65, 0x6e, 0x73, 0x6f, 0x72, 0x0, // "sensor\0"
				// label values pool
				0x73, 0x6f, 0x6c, 0x6f, 0x6d, 0x6f, 0x6e, 0x0, // "solomon\0"
				0x6d, 0x79, 0x67, 0x61, 0x75, 0x67, 0x65, 0x0, // "mygauge\0"
			},
			[]byte{0x0, 0x0, 0x0, 0x0}, // common time
			commonLabels{
				0x1,
				[][]byte{
					{
						0x0, // index of "project" in names pool
						0x0, // index of "solomon" in values pool
					},
				},
			},
			[][]byte{
				{
					0x5, // types (gauge without timestamp)
					0x0, // flags
					0x1, // labels count (sensor)
					0x1, // index of "sensor" in names pool
					0x1, // index of "mygauge" in values pool

					0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x45, 0x40, // 43: metrics value
				},
			},
			75,
		},
		{
			"gauge+gauge+repeating_commonLabels",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewGauge("mygauge", 42, WithTags(map[string]string{
							"host": "server1",
						}))
						return &g
					}(),
					func() Metric {
						g := NewGauge("othergauge", 100, WithTags(map[string]string{
							"dc": "man",
						}))
						return &g
					}(),
				},
				commonLabels: map[string]string{
					"host": "server1",
					"dc":   "man",
				},
			},
			version11,
			[]byte{
				0x53, 0x50, // magic
				0x01, 0x01, // version
				0x18, 0x00, // header size
				0x0,                 // time precision
				0x0,                 // compression algorithm
				0x0f, 0x0, 0x0, 0x0, // label names size (15 bytes: "host\0sensor\0dc\0")
				0x1f, 0x0, 0x0, 0x0, // label values size (31 bytes: "mygauge\0othergauge\0man\0server1\0")
				0x2, 0x0, 0x0, 0x0, // metric count
				0x2, 0x0, 0x0, 0x0, // point count
				// label names pool
				0x68, 0x6f, 0x73, 0x74, 0x0, // "host\0"
				0x64, 0x63, 0x0, // "dc\0"
				0x73, 0x65, 0x6e, 0x73, 0x6f, 0x72, 0x0, // "sensor\0"
				// label values pool
				0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x31, 0x0, // "server1\0"
				0x6d, 0x61, 0x6e, 0x0, // "man\0"
				0x6d, 0x79, 0x67, 0x61, 0x75, 0x67, 0x65, 0x0, // "mygauge\0"
				0x6f, 0x74, 0x68, 0x65, 0x72, 0x67, 0x61, 0x75, 0x67, 0x65, 0x0, // "othergauge\0"
			},
			[]byte{0x0, 0x0, 0x0, 0x0},
			commonLabels{
				0x2, // common labels count (2 labels)
				[][]byte{
					{
						0x1, // index of "dc" in names pool
						0x1, // index of "man" in values pool
					},
					{
						0x0, // index of "host" in names pool
						0x0, // index of "server1" in values pool
					},
				},
			},
			[][]byte{
				{
					0x5, // types (gauge without timestamp)
					0x0, // flags
					0x1, // labels count (only sensor)
					0x2, // index of "sensor" in names pool
					0x3, // index of "othergauge" in values pool

					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x59, 0x40, // 100: metrics value
				},
				{
					0x5, // types (gauge without timestamp)
					0x0, // flags
					0x1, // labels count (only sensor)
					0x2, // index of "sensor" in names pool
					0x2, // index of "mygauge" in values pool

					0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x45, 0x40, // 42: metrics value
				},
			},
			105,
		},
		{
			"counter+nameTag+existingNameLabel-v1.1",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewCounter(
							"mycounter",
							42,
							WithUseNameTag(),
							WithTags(map[string]string{
								"name": "custom_name_value",
							}),
						)
						return &g
					}(),
				},
			},
			version11,
			[]byte{
				0x53, 0x50, // magic
				0x01, 0x01, // version
				0x18, 0x00, // header size
				0x0,                // time precision
				0x0,                // compression algorithm
				0x5, 0x0, 0x0, 0x0, // label names size (5 bytes: "name\0")
				0x12, 0x0, 0x0, 0x0, // label values size (18 bytes: "custom_name_value\0")
				0x1, 0x0, 0x0, 0x0, // metric count
				0x1, 0x0, 0x0, 0x0, // point count
				// label names pool
				0x6e, 0x61, 0x6d, 0x65, 0x0, // "name\0"
				// label values pool
				0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x5f,
				0x76, 0x61, 0x6c, 0x75, 0x65, 0x0, // "custom_name_value\0"
			},
			[]byte{0x0, 0x0, 0x0, 0x0}, // common time
			commonLabels{
				0x0,
				[][]byte{},
			},
			[][]byte{
				{
					0x9,                                     // types (counter without timestamp)
					0x0,                                     // flags
					0x1,                                     // labels count (name)
					0x0,                                     // index of "name" in names pool
					0x0,                                     // index of "custom_name_value" in values pool
					0x2a, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // 42: metrics value
				},
			},
			65,
		},
		{
			"counter+nameTag+existingNameLabel-withUseNameTag-v1.2",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewCounter(
							"mycounter",
							42,
							WithUseNameTag(),
							WithTags(map[string]string{
								"name": "custom_name_value",
							}),
						)
						return &g
					}(),
				},
			},
			version12,
			[]byte{
				0x53, 0x50, // magic
				0x02, 0x01, // version
				0x18, 0x00, // header size
				0x0,                // time precision
				0x0,                // compression algorithm
				0x0, 0x0, 0x0, 0x0, // label names size (0 bytes)
				0x12, 0x0, 0x0, 0x0, // label values size (18 bytes: "custom_name_value\0")
				0x1, 0x0, 0x0, 0x0, // metric count
				0x1, 0x0, 0x0, 0x0, // point count
				// label names pool
				// empty
				// label values pool
				0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x5f,
				0x76, 0x61, 0x6c, 0x75, 0x65, 0x0, // "custom_name_value\0"
			},
			[]byte{0x0, 0x0, 0x0, 0x0}, // common time
			commonLabels{
				0x0,
				[][]byte{},
			},
			[][]byte{
				{
					0x9, // types (counter without timestamp)
					0x0, // flags
					0x0, // nameValueIndex
					0x0, // labels count (name)

					0x2a, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // 42: metrics value
				},
			},
			59,
		},
		{
			"counter+nameTag+existingNameLabel-v1.2",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewCounter(
							"mycounter",
							42,
							WithTags(map[string]string{
								"name": "custom_name_value",
							}),
						)
						return &g
					}(),
				},
			},
			version12,
			[]byte{
				0x53, 0x50, // magic
				0x02, 0x01, // version
				0x18, 0x00, // header size
				0x0,                // time precision
				0x0,                // compression algorithm
				0x0, 0x0, 0x0, 0x0, // label names size (0 bytes)
				0x12, 0x0, 0x0, 0x0, // label values size (18 bytes: "custom_name_value\0")
				0x1, 0x0, 0x0, 0x0, // metric count
				0x1, 0x0, 0x0, 0x0, // point count
				// label names pool
				// empty
				// label values pool
				0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x5f,
				0x76, 0x61, 0x6c, 0x75, 0x65, 0x0, // "custom_name_value\0"
			},
			[]byte{0x0, 0x0, 0x0, 0x0}, // common time
			commonLabels{
				0x0,
				[][]byte{},
			},
			[][]byte{
				{
					0x9, // types (counter without timestamp)
					0x0, // flags
					0x0, // nameValueIndex
					0x0, // labels count (name)

					0x2a, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // 42: metrics value
				},
			},
			59,
		},
		{
			"counter-v1.2",
			&Metrics{
				metrics: []Metric{
					func() Metric {
						g := NewCounter("mycounter", 42)
						return &g
					}(),
				},
			},
			version12,
			[]byte{
				0x53, 0x50, // magic
				0x02, 0x01, // version
				0x18, 0x00, // header size
				0x0,                // time precision
				0x0,                // compression algorithm
				0x0, 0x0, 0x0, 0x0, // label names size (0 bytes)
				0xa, 0x0, 0x0, 0x0, // label values size (10 bytes: "mycounter\0")
				0x1, 0x0, 0x0, 0x0, // metric count
				0x1, 0x0, 0x0, 0x0, // point count
				// label names pool
				// label values pool
				0x6d, 0x79, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x65, 0x72, 0x0, // "mycounter\0"
			},
			[]byte{0x0, 0x0, 0x0, 0x0}, // common time
			commonLabels{
				0x0,
				[][]byte{},
			},
			[][]byte{
				{
					0x9,                                     // types (counter without timestamp)
					0x0,                                     // flags
					0x0,                                     // nameValueIndex (должен указывать на "mycounter")
					0x0,                                     // labels count
					0x2a, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, // 42: metrics value
				},
			},
			51,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := context.Background()

			encoder := NewSpackEncoder(ctx, CompressionNone, tc.metrics)

			encoder.version = tc.version

			written, err := encoder.Encode(&buf)

			assert.NoError(t, err)
			assert.Equal(t, tc.expectWritten, written)

			body := buf.Bytes()
			setMetricsCount(tc.expectHeader, len(tc.metrics.metrics))

			require.True(t, bytes.HasPrefix(body, tc.expectHeader[:HeaderSize]))
			body = body[HeaderSize:]

			nameSize := int(binary.LittleEndian.Uint32(tc.expectHeader[8:12]))
			valueSize := int(binary.LittleEndian.Uint32(tc.expectHeader[12:16]))
			expectPools := tc.expectHeader[HeaderSize:]
			require.Len(t, expectPools, nameSize+valueSize)

			require.ElementsMatch(
				t,
				bytes.Split(expectPools[:nameSize], []byte{0}),
				bytes.Split(body[:nameSize], []byte{0}),
				"label name pool mismatch",
			)
			require.ElementsMatch(
				t,
				bytes.Split(expectPools[nameSize:], []byte{0}),
				bytes.Split(body[nameSize:nameSize+valueSize], []byte{0}),
				"label value pool mismatch",
			)
			body = body[nameSize+valueSize:]

			require.True(t, bytes.HasPrefix(body, tc.expectCommonTime))
			body = body[len(tc.expectCommonTime):]

			require.True(t, body[0] == tc.expectCommonLabels.count)
			body = body[1:]

			for _, l := range tc.expectCommonLabels.labels {
				require.True(t, bytes.Contains(body, l))
				body = bytes.Replace(body, l, []byte{}, 1)
			}

			var expectButMissing [][]byte
			for _, v := range tc.expectMetrics {
				if bytes.Contains(body, v) {
					body = bytes.Replace(body, v, []byte{}, 1)
				} else {
					expectButMissing = append(expectButMissing, v)
				}
			}
			assert.Empty(t, body, "unexpected bytes seen")
			assert.Empty(t, expectButMissing, "missing metrics bytes")
		})
	}
}

func setMetricsCount(header []byte, count int) {
	header[16] = uint8(count)
	header[20] = uint8(count)
}

func makeBenchGauges(n int) []Metric {
	out := make([]Metric, n)
	for i := 0; i < n; i++ {
		g := NewGauge(
			fmt.Sprintf("mygauge_%d", i),
			float64(i),
			WithTags(map[string]string{
				"host":    fmt.Sprintf("host-%d", i%16),
				"dc":      []string{"man", "vla", "sas", "iva"}[i%4],
				"service": fmt.Sprintf("svc-%d", i%8),
			}),
		)
		out[i] = &g
	}
	return out
}

func makeBenchCounters(n int) []Metric {
	out := make([]Metric, n)
	for i := 0; i < n; i++ {
		c := NewCounter(
			fmt.Sprintf("mycounter_%d", i),
			int64(i),
			WithTags(map[string]string{
				"host":   fmt.Sprintf("host-%d", i%16),
				"region": []string{"ru-central1", "ru-central2", "ru-central3"}[i%3],
			}),
		)
		out[i] = &c
	}
	return out
}

func makeBenchHistograms(n int) []Metric {
	bounds := []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1, 5, 10, 50}
	out := make([]Metric, n)
	for i := 0; i < n; i++ {
		values := make([]int64, len(bounds))
		for j := range values {
			values[j] = int64((i + j) * 7)
		}
		h := NewHistogram(
			fmt.Sprintf("myhist_%d", i),
			bounds,
			values,
			int64(i),
			WithTags(map[string]string{
				"host":     fmt.Sprintf("host-%d", i%16),
				"endpoint": fmt.Sprintf("/api/v1/path%d", i%32),
			}),
		)
		out[i] = &h
	}
	return out
}

func makeBenchMixed(n int) []Metric {
	g := makeBenchGauges(n / 3)
	c := makeBenchCounters(n / 3)
	h := makeBenchHistograms(n - len(g) - len(c))
	out := make([]Metric, 0, n)
	out = append(out, g...)
	out = append(out, c...)
	out = append(out, h...)
	return out
}

func benchmarkSpackEncode(b *testing.B, metrics *Metrics, compression CompressionType, opts ...SpackOpts) {
	ctx := context.Background()
	b.ReportAllocs()
	for b.Loop() {
		enc := NewSpackEncoder(ctx, compression, metrics, opts...)
		if _, err := enc.Encode(io.Discard); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSpackEncode_Gauges(b *testing.B) {
	for _, size := range []int{1, 10, 100, 1000, 10000} {
		m := &Metrics{metrics: makeBenchGauges(size)}
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone)
		})
	}
}

func BenchmarkSpackEncode_Counters(b *testing.B) {
	for _, size := range []int{1, 10, 100, 1000, 10000} {
		m := &Metrics{metrics: makeBenchCounters(size)}
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone)
		})
	}
}

func BenchmarkSpackEncode_Histograms(b *testing.B) {
	for _, size := range []int{1, 10, 100, 1000} {
		m := &Metrics{metrics: makeBenchHistograms(size)}
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone)
		})
	}
}

func BenchmarkSpackEncode_Mixed(b *testing.B) {
	for _, size := range []int{30, 300, 3000} {
		m := &Metrics{metrics: makeBenchMixed(size)}
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone)
		})
	}
}

func BenchmarkSpackEncode_CommonLabels(b *testing.B) {
	const size = 1000
	m := &Metrics{
		metrics: makeBenchGauges(size),
		commonLabels: map[string]string{
			"project": "solomon",
			"cluster": "production",
			"env":     "prod",
		},
	}
	benchmarkSpackEncode(b, m, CompressionNone)
}

func BenchmarkSpackEncode_Version12(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		m := &Metrics{metrics: makeBenchGauges(size)}
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone, WithVersion12())
		})
	}
}

func BenchmarkSpackEncode_LZ4(b *testing.B) {
	for _, size := range []int{100, 1000, 10000} {
		m := &Metrics{metrics: makeBenchGauges(size)}
		b.Run(fmt.Sprintf("size=%d", size), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionLz4)
		})
	}
}

func BenchmarkSpackEncode_HighLabelCardinality(b *testing.B) {
	const size = 1000
	out := make([]Metric, size)
	for i := 0; i < size; i++ {
		tags := make(map[string]string, 8)
		for j := 0; j < 8; j++ {
			tags[fmt.Sprintf("label_%d_%d", i, j)] = fmt.Sprintf("value_%d_%d", i, j)
		}
		g := NewGauge(fmt.Sprintf("metric_%d", i), float64(i), WithTags(tags))
		out[i] = &g
	}
	m := &Metrics{metrics: out}
	benchmarkSpackEncode(b, m, CompressionNone)
}

func BenchmarkSpackEncode_SharedLabels(b *testing.B) {
	const size = 1000
	sharedTags := map[string]string{
		"host":    "host-0",
		"dc":      "man",
		"service": "svc-0",
		"env":     "prod",
	}
	out := make([]Metric, size)
	for i := 0; i < size; i++ {
		g := NewGauge(fmt.Sprintf("metric_%d", i), float64(i), WithTags(sharedTags))
		out[i] = &g
	}
	m := &Metrics{metrics: out}
	benchmarkSpackEncode(b, m, CompressionNone)
}

func makeBenchGaugePoints(n int) []GaugePoint {
	points := make([]GaugePoint, n)
	base := time.Unix(1700000000, 0)
	for i := 0; i < n; i++ {
		points[i] = GaugePoint{
			Timestamp: base.Add(time.Duration(i) * time.Second),
			Value:     float64(i) * 0.5,
		}
	}
	return points
}

func makeBenchCounterPoints(n int) []CounterPoint {
	points := make([]CounterPoint, n)
	base := time.Unix(1700000000, 0)
	for i := 0; i < n; i++ {
		points[i] = CounterPoint{
			Timestamp: base.Add(time.Duration(i) * time.Second),
			Value:     int64(i),
		}
	}
	return points
}

func makeBenchIGaugePoints(n int) []IGaugePoint {
	points := make([]IGaugePoint, n)
	base := time.Unix(1700000000, 0)
	for i := 0; i < n; i++ {
		points[i] = IGaugePoint{
			Timestamp: base.Add(time.Duration(i) * time.Second),
			Value:     int64(i) * 3,
		}
	}
	return points
}

func makeBenchGaugeSeries(numMetrics, pointsPerSeries int) []Metric {
	out := make([]Metric, numMetrics)
	for i := 0; i < numMetrics; i++ {
		s := NewGaugeSeries(
			fmt.Sprintf("mygauge_%d", i),
			makeBenchGaugePoints(pointsPerSeries),
			WithTags(map[string]string{
				"host":    fmt.Sprintf("host-%d", i%16),
				"dc":      []string{"man", "vla", "sas", "iva"}[i%4],
				"service": fmt.Sprintf("svc-%d", i%8),
			}),
		)
		out[i] = &s
	}
	return out
}

func makeBenchCounterSeries(numMetrics, pointsPerSeries int) []Metric {
	out := make([]Metric, numMetrics)
	for i := 0; i < numMetrics; i++ {
		s := NewCounterSeries(
			fmt.Sprintf("mycounter_%d", i),
			makeBenchCounterPoints(pointsPerSeries),
			WithTags(map[string]string{
				"host":   fmt.Sprintf("host-%d", i%16),
				"region": []string{"ru-central1", "ru-central2", "ru-central3"}[i%3],
			}),
		)
		out[i] = &s
	}
	return out
}

func makeBenchIGaugeSeries(numMetrics, pointsPerSeries int) []Metric {
	out := make([]Metric, numMetrics)
	for i := 0; i < numMetrics; i++ {
		s := NewIGaugeSeries(
			fmt.Sprintf("myigauge_%d", i),
			makeBenchIGaugePoints(pointsPerSeries),
			WithTags(map[string]string{
				"host":     fmt.Sprintf("host-%d", i%16),
				"endpoint": fmt.Sprintf("/api/v1/path%d", i%32),
			}),
		)
		out[i] = &s
	}
	return out
}

func makeBenchMixedSeries(numMetrics, pointsPerSeries int) []Metric {
	g := makeBenchGaugeSeries(numMetrics/3, pointsPerSeries)
	c := makeBenchCounterSeries(numMetrics/3, pointsPerSeries)
	i := makeBenchIGaugeSeries(numMetrics-len(g)-len(c), pointsPerSeries)
	out := make([]Metric, 0, numMetrics)
	out = append(out, g...)
	out = append(out, c...)
	out = append(out, i...)
	return out
}

var benchSeriesShapes = []struct {
	numMetrics      int
	pointsPerSeries int
}{
	{10, 10},
	{100, 10},
	{1000, 10},
	{100, 60},
	{100, 600},
	{1000, 60},
}

func BenchmarkSpackEncode_GaugeSeries(b *testing.B) {
	for _, s := range benchSeriesShapes {
		m := &Metrics{metrics: makeBenchGaugeSeries(s.numMetrics, s.pointsPerSeries)}
		b.Run(fmt.Sprintf("metrics=%d/points=%d", s.numMetrics, s.pointsPerSeries), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone)
		})
	}
}

func BenchmarkSpackEncode_CounterSeries(b *testing.B) {
	for _, s := range benchSeriesShapes {
		m := &Metrics{metrics: makeBenchCounterSeries(s.numMetrics, s.pointsPerSeries)}
		b.Run(fmt.Sprintf("metrics=%d/points=%d", s.numMetrics, s.pointsPerSeries), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone)
		})
	}
}

func BenchmarkSpackEncode_IGaugeSeries(b *testing.B) {
	for _, s := range benchSeriesShapes {
		m := &Metrics{metrics: makeBenchIGaugeSeries(s.numMetrics, s.pointsPerSeries)}
		b.Run(fmt.Sprintf("metrics=%d/points=%d", s.numMetrics, s.pointsPerSeries), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone)
		})
	}
}

func BenchmarkSpackEncode_MixedSeries(b *testing.B) {
	for _, s := range benchSeriesShapes {
		m := &Metrics{metrics: makeBenchMixedSeries(s.numMetrics, s.pointsPerSeries)}
		b.Run(fmt.Sprintf("metrics=%d/points=%d", s.numMetrics, s.pointsPerSeries), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone)
		})
	}
}

func BenchmarkSpackEncode_GaugeSeries_LZ4(b *testing.B) {
	for _, s := range benchSeriesShapes {
		m := &Metrics{metrics: makeBenchGaugeSeries(s.numMetrics, s.pointsPerSeries)}
		b.Run(fmt.Sprintf("metrics=%d/points=%d", s.numMetrics, s.pointsPerSeries), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionLz4)
		})
	}
}

func BenchmarkSpackEncode_GaugeSeries_Version12(b *testing.B) {
	for _, s := range benchSeriesShapes {
		m := &Metrics{metrics: makeBenchGaugeSeries(s.numMetrics, s.pointsPerSeries)}
		b.Run(fmt.Sprintf("metrics=%d/points=%d", s.numMetrics, s.pointsPerSeries), func(b *testing.B) {
			benchmarkSpackEncode(b, m, CompressionNone, WithVersion12())
		})
	}
}
