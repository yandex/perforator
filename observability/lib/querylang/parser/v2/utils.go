package parserv2

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

var solomonDurationRegex = regexp.MustCompile(`^((\d+)((ms)|w|d|h|m|s))+$`)
var solomonDurationPart = regexp.MustCompile(`(\d+)((ms)|w|d|h|m|s)`)

// parse a duration string in the format of Solomon.
// See https://github.com/yandex/perforator/arcadia/solomon/libs/java/solomon-grammar/SolomonLexer.g4?rev=r8891503#L62
func ParseSolomonDuration(duration string) (time.Duration, error) {
	if !solomonDurationRegex.MatchString(duration) {
		return 0, fmt.Errorf("not match duration regex")
	}
	match := solomonDurationPart.FindAllStringSubmatch(duration, -1)
	if match == nil {
		return 0, fmt.Errorf("something went wrong. Not match duration parts")
	}
	result := int64(0)
	for _, durationPart := range match {

		val, err := strconv.ParseInt(durationPart[1], 10, 0)
		if err != nil {
			return 0, fmt.Errorf("cannot parse value %q: %w", durationPart[1], err)
		}

		unit, ok := unitMap[durationPart[2]]
		if !ok {
			return 0, fmt.Errorf("unknown unit %q", durationPart[2])
		}

		result += val * unit
	}

	return time.Duration(result), nil
}

var unitMap = map[string]int64{
	"ms": int64(time.Millisecond),
	"s":  int64(time.Second),
	"m":  int64(time.Minute),
	"h":  int64(time.Hour),
	"d":  int64(time.Hour) * 24,
	"w":  int64(time.Hour) * 24 * 7,
}
