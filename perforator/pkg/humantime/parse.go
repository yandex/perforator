package humantime

import (
	"errors"
	"fmt"
	"time"
)

var parsers = []func(string, *time.Location) (time.Time, error){
	parseUnixTimestamp,
	parseWellKnownTimeLayouts,
	parseSpecialTimes,
	parseLast24HoursTime,
	parseSpecialTimes,
	parseRelativeTime,
}

func Parse(s string) (time.Time, error) {
	return ParseInLocation(s, time.Local)
}

func ParseInLocation(s string, loc *time.Location) (time.Time, error) {
	errs := make([]error, 0)

	for _, parser := range parsers {
		ts, err := parser(s, loc)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		return ts, nil
	}

	return time.Time{}, fmt.Errorf("failed to parse time, errs: %w", errors.Join(errs...))
}

func ParseInterval(begin string, end string) (
	st time.Time,
	et time.Time,
	err error,
) {
	if begin != "" {
		st, err = Parse(begin)
		if err != nil {
			return
		}
	}

	et = now()
	if end != "" {
		et, err = Parse(end)
		if err != nil {
			return
		}
	}

	if st.After(et) {
		err = fmt.Errorf(
			"start time must be less than end time, parsed: %s %s",
			st.String(),
			et.String(),
		)
		return
	}

	return
}
