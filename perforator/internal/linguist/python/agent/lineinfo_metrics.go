package agent

import (
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
)

// Count only ordinary symbolized frames with line information enabled.
// Resolved includes cached lines; failed includes negative cache entries.
// NoLine means a table was obtained but yielded no positive source line.
type lineInfoOutcome = models.LineInfoOutcome

const (
	lineSkipped          = models.LineInfoSkipped
	lineResolved         = models.LineInfoResolved
	lineUnavailable      = models.LineInfoUnavailable
	lineNoLine           = models.LineInfoNoLine
	lineFailed           = models.LineInfoFailed
	lineInfoOutcomeCount = lineFailed + 1
)

type lineInfoMetrics struct {
	frames [lineInfoOutcomeCount]metrics.Counter
}

func newLineInfoMetrics(reg metrics.Registry, enabled bool) lineInfoMetrics {
	reg = reg.WithPrefix("python.lineinfo")
	var enabledValue float64
	if enabled {
		enabledValue = 1
	}
	reg.Gauge("enabled").Set(enabledValue)

	var m lineInfoMetrics
	for outcome, result := range [lineInfoOutcomeCount]string{
		lineResolved:    "resolved",
		lineUnavailable: "unavailable",
		lineNoLine:      "no_line",
		lineFailed:      "failed",
	} {
		if lineInfoOutcome(outcome) != lineSkipped {
			m.frames[outcome] = reg.WithTags(map[string]string{"result": result}).Counter("frames.count")
		}
	}
	return m
}

// Accumulate locally to avoid a metric update for every frame in a sample.
func (m *lineInfoMetrics) record(counts *[lineInfoOutcomeCount]int64) {
	for outcome := lineResolved; outcome < lineInfoOutcomeCount; outcome++ {
		if count := counts[outcome]; count != 0 {
			m.frames[outcome].Add(count)
		}
	}
}
