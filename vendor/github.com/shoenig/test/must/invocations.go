// Code generated via scripts/generate.sh. DO NOT EDIT.

// Copyright (c) The Test Authors
// SPDX-License-Identifier: MPL-2.0

// Package test provides a modern generic testing assertions library.
package must

import (
	"strings"

	"github.com/shoenig/test/internal/assertions"
)

func passing(result string) bool {
	return result == ""
}

func fail(t T, msg string, scripts ...PostScript) {
	c := assertions.Caller()
	s := c + msg + "\n" + run(scripts...)
	errorf(t, "\n"+strings.TrimSpace(s)+"\n")
}

func invoke(t T, result string, settings ...Setting) {
	result = strings.TrimSpace(result)
	if !passing(result) {
		fail(t, result, scripts(settings...)...)
	}
}
