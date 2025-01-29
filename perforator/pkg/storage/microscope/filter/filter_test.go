package filter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
)

const (
	baseTimeUnix = int64(1719558381)
)

func TestFilter_SegmentScanline(t *testing.T) {
	filter := NewMapLabelFilter(PodFilter)
	filter.AddValue("1", time.Unix(baseTimeUnix+3, 0), time.Unix(baseTimeUnix+5, 0))
	filter.AddValue("1", time.Unix(baseTimeUnix, 0), time.Unix(baseTimeUnix+2, 0))
	filter.AddValue("1", time.Unix(baseTimeUnix+6, 0), time.Unix(baseTimeUnix+10, 0))
	filter.AddValue("1", time.Unix(baseTimeUnix+8, 0), time.Unix(baseTimeUnix+13, 0))
	filter.AddValue("1", time.Unix(baseTimeUnix+8, 0), time.Unix(baseTimeUnix+9, 0))

	filter.Finalize()

	tests := map[time.Time]bool{
		time.Unix(baseTimeUnix-1, 0):      false,
		time.Unix(baseTimeUnix, 0):        true,
		time.Unix(baseTimeUnix+1, 0):      true,
		time.Unix(baseTimeUnix+2, 0):      true,
		time.Unix(baseTimeUnix+2, 123123): false,
		time.Unix(baseTimeUnix+4, 0):      true,
		time.Unix(baseTimeUnix+5, 123123): false,
		time.Unix(baseTimeUnix+6, 0):      true,
		time.Unix(baseTimeUnix+7, 0):      true,
		time.Unix(baseTimeUnix+10, 0):     true,
		time.Unix(baseTimeUnix+12, 0):     true,
		time.Unix(baseTimeUnix+15, 0):     false,
	}

	for ts, res := range tests {
		require.Equal(
			t, res, filter.Filter(&meta.ProfileMetadata{Timestamp: ts, PodID: "1"}),
			"Failed filter for relative point %d, relative nanos %d",
			ts.Unix()-baseTimeUnix,
			ts.UnixNano()-time.Unix(baseTimeUnix, 0).UnixNano(),
		)
	}
}

func TestFilter_MultipleValues(t *testing.T) {
	filter := NewMapLabelFilter(NodeFilter)
	filter.AddValue("3", time.Unix(baseTimeUnix+1, 0), time.Unix(baseTimeUnix+3, 0))
	filter.AddValue("4", time.Unix(baseTimeUnix+1, 0), time.Unix(baseTimeUnix+3, 0))
	filter.AddValue("1", time.Unix(baseTimeUnix, 0), time.Unix(baseTimeUnix+3, 0))
	filter.AddValue("2", time.Unix(baseTimeUnix, 0), time.Unix(baseTimeUnix+3, 0))
	filter.AddValue("1", time.Unix(baseTimeUnix-1, 0), time.Unix(baseTimeUnix+1, 0))
	filter.Finalize()

	tests := map[*meta.ProfileMetadata]bool{
		&meta.ProfileMetadata{
			NodeID:    "3",
			Timestamp: time.Unix(baseTimeUnix+2, 0),
		}: true,
		&meta.ProfileMetadata{
			NodeID:    "3",
			Timestamp: time.Unix(baseTimeUnix, 0),
		}: false,
		&meta.ProfileMetadata{
			NodeID:    "1",
			Timestamp: time.Unix(baseTimeUnix, -123123),
		}: true,
		&meta.ProfileMetadata{
			NodeID:    "10",
			Timestamp: time.Unix(baseTimeUnix, -123123),
		}: false,
		&meta.ProfileMetadata{
			NodeID:    "4",
			Timestamp: time.Unix(baseTimeUnix+10, 0),
		}: false,
	}

	for meta, res := range tests {
		require.Equal(
			t, res, filter.Filter(meta),
		)
	}
}
