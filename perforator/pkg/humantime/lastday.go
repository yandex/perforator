package humantime

import (
	"fmt"
	"time"
)

func parseLast24HoursTime(hhmm string, loc *time.Location) (time.Time, error) {
	var h, m int
	n, err := fmt.Sscanf(hhmm, "%d:%d", &h, &m)
	if err != nil || n != 2 {
		return time.Time{}, fmt.Errorf("cannot parse HH:MM: %w", err)
	}

	now := now()
	t := time.Date(now.Year(), now.Month(), now.Day(), h, m, 0, 0, loc)

	if t.After(now) {
		// subtract one day
		t = t.AddDate(0, 0, -1)
	}

	return t, nil
}
