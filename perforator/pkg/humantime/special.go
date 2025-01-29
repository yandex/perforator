package humantime

import (
	"fmt"
	"regexp"
	"time"
)

const (
	LongTimeAgo = "a long time ago"
	Now         = "now"
)

func parseSpecialTimes(s string, _ *time.Location) (t time.Time, err error) {
	switch s {
	case "now":
		t = now()
	case "a long time ago", "zero", "null":
		t = time.Time{}
	default:
		err = fmt.Errorf("failed to parse special time")
	}
	return
}

var relativeRegex = regexp.MustCompile(`^now\s*-\s*(.*)$`)

func parseRelativeTime(s string, _ *time.Location) (t time.Time, err error) {
	groups := relativeRegex.FindStringSubmatch(s)
	if groups == nil || len(groups) != 2 {
		err = fmt.Errorf("failed to parse relative time")
		return
	}

	delta, err := time.ParseDuration(groups[1])
	if err != nil {
		return time.Time{}, err
	}

	t = now().Add(-delta)
	return
}
