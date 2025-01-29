package humantime

import (
	"errors"
	"fmt"
	"time"
)

var layouts = []string{
	time.RFC3339,

	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02",

	"02.01.06 15:04:05",
	"02.01.06 15:04",
	"02.01.06",
}

func parseWellKnownTimeLayouts(input string, loc *time.Location) (time.Time, error) {
	errs := make([]error, 0)

	for _, layout := range layouts {
		res, err := time.ParseInLocation(layout, input, loc)
		if err == nil {
			return res, nil
		}
		errs = append(errs, fmt.Errorf("layout %q: %w", layout, err))
	}

	return time.Time{}, errors.Join(errs...)
}
