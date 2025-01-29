package humantime

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/ptr"
)

func TestHumanTimeParse(t *testing.T) {
	for _, test := range []struct {
		name     string
		input    string
		expected time.Time
		now      *time.Time
		err      string
	}{
		{
			"last day",
			"12:23",
			time.Date(2042, 11, 9, 12, 23, 0, 0, time.UTC),
			ptr.T(time.Date(2042, 11, 9, 22, 05, 13, 14, time.UTC)),
			"",
		},
		{
			"previous day",
			"12:23",
			time.Date(2042, time.November, 8, 12, 23, 0, 0, time.UTC),
			ptr.T(time.Date(2042, time.November, 9, 10, 5, 13, 14, time.UTC)),
			"",
		},
		{
			"previous minute",
			"23:59",
			time.Date(2042, time.November, 8, 23, 59, 0, 0, time.UTC),
			ptr.T(time.Date(2042, time.November, 9, 0, 0, 0, 1, time.UTC)),
			"",
		},
		{
			"special now",
			"now",
			time.Date(1337, time.January, 24, 10, 15, 0, 1, time.UTC),
			ptr.T(time.Date(1337, time.January, 24, 10, 15, 0, 1, time.UTC)),
			"",
		},
		{
			"special a long time ago",
			"a long time ago",
			time.Time{},
			ptr.T(time.Date(1337, time.January, 24, 10, 15, 0, 1, time.UTC)),
			"",
		},
		{
			"unix ts",
			"1691622800",
			time.Date(2023, time.August, 9, 23, 13, 20, 0, time.UTC),
			ptr.T(time.Date(1337, time.January, 24, 10, 15, 0, 1, time.UTC)),
			"",
		},
		{
			"error",
			"nowww",
			time.Time{},
			ptr.T(time.Date(1337, time.January, 24, 10, 15, 0, 1, time.UTC)),
			"failed to parse time",
		},
		{
			"well-known 1",
			"2023-03-29 19:56",
			time.Date(2023, time.March, 29, 19, 56, 0, 0, time.Local),
			nil,
			"",
		},
		{
			"well-known 2",
			"2023-07-15",
			time.Date(2023, time.July, 15, 0, 0, 0, 0, time.Local),
			nil,
			"",
		},
		{
			"relative",
			"now-5h2m1ns",
			time.Date(1337, time.January, 24, 5, 13, 7, 1, time.UTC),
			ptr.T(time.Date(1337, time.January, 24, 10, 15, 7, 2, time.UTC)),
			"",
		},
		{
			"relative with spaces 1",
			"now         - 5h2m1ns",
			time.Date(1337, time.January, 24, 5, 13, 7, 1, time.UTC),
			ptr.T(time.Date(1337, time.January, 24, 10, 15, 7, 2, time.UTC)),
			"",
		},
		{
			"relative with spaces 2",
			"now         -5h2m1ns",
			time.Date(1337, time.January, 24, 5, 13, 7, 1, time.UTC),
			ptr.T(time.Date(1337, time.January, 24, 10, 15, 7, 2, time.UTC)),
			"",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.now != nil {
				now = func() time.Time {
					return *test.now
				}
			} else {
				now = time.Now
			}

			ts, err := Parse(test.input)
			if test.err == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, test.err)
				return
			}

			require.Equal(t, test.expected.UTC(), ts.UTC())
		})
	}
}
