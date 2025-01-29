package util

import (
	"time"
)

type (
	Pagination struct {
		Offset uint64
		Limit  uint64
	}

	TimeRange struct {
		FromTimestamp *time.Time
		ToTimestamp   *time.Time
	}
)
