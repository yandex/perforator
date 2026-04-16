// Copyright (c) The Test Authors
// SPDX-License-Identifier: MPL-2.0

package must

// T is the minimal set of functions to be implemented by any testing framework
// compatible with the must package.
type T interface {
	Helper()
	Fatalf(string, ...any)
}

func errorf(t T, msg string, args ...any) {
	t.Fatalf(msg, args...)
}
