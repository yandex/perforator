package mock

import (
	"time"

	"go.uber.org/atomic"

	"github.com/yandex/perforator/library/go/core/metrics"
)

var _ metrics.Timer = (*Timer)(nil)

// Timer measures gauge duration.
type Timer struct {
	Name  string
	Tags  map[string]string
	Value *atomic.Duration
}

func (t *Timer) RecordDuration(value time.Duration) {
	t.Value.Store(value)
}
