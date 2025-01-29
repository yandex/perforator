package perfmap

import (
	"fmt"
	"strconv"
	"strings"
)

type processConfig struct {
	percentage uint32
	java       bool
}

func parseProcessConfig(config string) (*processConfig, []error) {
	var errs []error
	parts := strings.Split(config, ",")
	parsed := &processConfig{
		percentage: 100,
	}
	for _, part := range parts {
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			errs = append(errs, fmt.Errorf("config part is not key-value pair: %s", part))
			continue
		}
		switch kv[0] {
		case "percentage":
			percentage, err := strconv.ParseUint(kv[1], 10, 32)
			if err != nil {
				errs = append(errs, fmt.Errorf("invalid percentage field: value it not number %q", kv[1]))
				continue
			}
			if percentage > 100 {
				errs = append(errs, fmt.Errorf("invalid percentage field: value %d is greater than 100", percentage))
				continue
			}
			parsed.percentage = uint32(percentage)
		case "java":
			if kv[1] != "true" {
				errs = append(errs, fmt.Errorf("invalid java field: only true is supported, got %q", kv[1]))
				continue
			}
			parsed.java = true
		default:
			errs = append(errs, fmt.Errorf("invalid config field: %q", kv[0]))
		}
	}

	return parsed, errs
}
