package humantime

import (
	"strconv"
	"time"
)

func parseUnixTimestamp(s string, _ *time.Location) (time.Time, error) {
	seconds, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(int64(seconds), 0), nil
}
