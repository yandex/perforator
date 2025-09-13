package agent

import (
	"context"
	"os"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profiler"
)

type DebugModeTogglerConfig struct {
	Interval    time.Duration
	TogglerPath string
}

type debugModeTogglerWatcher struct {
	l        log.Logger
	conf     *DebugModeTogglerConfig
	profiler *profiler.Profiler
}

func newDebugModeTogglerWatcher(l log.Logger, conf *DebugModeTogglerConfig, profiler *profiler.Profiler) *debugModeTogglerWatcher {
	return &debugModeTogglerWatcher{
		l:        l,
		conf:     conf,
		profiler: profiler,
	}
}

func (d *debugModeTogglerWatcher) run(ctx context.Context) error {
	tick := time.NewTicker(d.conf.Interval)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-tick.C:
		}

		if _, err := os.Stat(d.conf.TogglerPath); err == nil {
			err = d.profiler.SetDebugMode(true)
			if err != nil {
				d.l.Error("Failed to enable debug mode", log.Error(err))
			}
		} else {
			err = d.profiler.SetDebugMode(false)
			if err != nil {
				d.l.Error("Failed to disable debug mode", log.Error(err))
			}
		}
	}
}
