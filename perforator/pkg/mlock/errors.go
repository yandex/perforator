package mlock

import "errors"

var ErrNotSupported = errors.New("failed to lock executable mappings: mlock is not supported on this platform")
