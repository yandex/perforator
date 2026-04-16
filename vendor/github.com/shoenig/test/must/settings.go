// Code generated via scripts/generate.sh. DO NOT EDIT.

// Copyright (c) The Test Authors
// SPDX-License-Identifier: MPL-2.0

package must

import (
	"github.com/google/go-cmp/cmp"
)

// Settings are used to manage a collection of Setting values used to modify
// the behavior of a test case assertion. Currently supports specifying custom
// error output content, and custom cmp.Option comparators / transforms.
//
// Use Cmp for specifying custom cmp.Option values.
//
// Use Sprint, Sprintf, Values, Func for specifying custom failure output messages.
type Settings struct {
	postScripts []PostScript
	cmpOptions  []cmp.Option
}

// A Setting changes the behavior of a test case assertion.
type Setting func(s *Settings)

// Cmp enables configuring cmp.Option values for modifying the behavior of the
// cmp.Equal function. Custom cmp.Option values control how the cmp.Equal function
// determines equality between the two objects.
//
// https://github.com/google/go-cmp/blob/master/cmp/options.go#L16
func Cmp(options ...cmp.Option) Setting {
	return func(s *Settings) {
		s.cmpOptions = append(s.cmpOptions, options...)
	}
}

func options(settings ...Setting) []cmp.Option {
	s := new(Settings)
	for _, setting := range settings {
		setting(s)
	}
	return s.cmpOptions
}

func scripts(settings ...Setting) []PostScript {
	s := new(Settings)
	for _, setting := range settings {
		setting(s)
	}
	return s.postScripts
}
