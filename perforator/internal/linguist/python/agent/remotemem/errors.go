package remotemem

import "errors"

// ErrCodeObjectChanged is returned by ReadCodeLinetable when the re-read
// co_firstlineno or co_linetable does not match the value captured at BPF
// sample time. Userspace should fall back to no line number.
var ErrCodeObjectChanged = errors.New("python: PyCodeObject changed since sampling")
