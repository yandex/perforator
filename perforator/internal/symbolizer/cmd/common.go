package cmd

import (
	"fmt"
	"strings"

	"github.com/yandex/perforator/perforator/pkg/profile/python"
)

func parsePythonPrettifyLevel(s string) (python.PrettifyLevel, error) {
	switch strings.ToLower(s) {
	case "", "off":
		return python.PrettifyOff, nil
	case "mixed":
		return python.PrettifyMixed, nil
	case "python-only":
		return python.PrettifyPythonOnly, nil
	default:
		return python.PrettifyOff, fmt.Errorf("unknown --experimental-prettify-python-stacks value %q: must be one of off, mixed, python-only", s)
	}
}
