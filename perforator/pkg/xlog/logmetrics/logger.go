package logmetrics

import (
	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
)

////////////////////////////////////////////////////////////////////////////////

func NewMeteredLogger(l log.Logger, r metrics.Registry) log.Logger {
	levels := log.Levels()
	maxlevel := 0
	for _, level := range levels {
		maxlevel = max(maxlevel, int(level))
	}
	counts := make([]metrics.Counter, maxlevel+1)
	for _, level := range levels {
		counts[int(level)] = r.
			WithPrefix("log").
			WithTags(map[string]string{"level": level.String()}).
			Counter("message.count")
	}

	return &logger{log.AddCallerSkip(l, 1), counts}
}

type logger struct {
	l            log.Logger
	countByLevel []metrics.Counter
}

var _ log.Logger = (*logger)(nil)
var _ log.LoggerWith = (*logger)(nil)
var _ log.LoggerAddCallerSkip = (*logger)(nil)

////////////////////////////////////////////////////////////////////////////////

// Logger implements log.Fmt.
func (l *logger) Logger() log.Logger {
	return l
}

// Fmt implements log.Logger.
func (l *logger) Fmt() log.Fmt {
	return l
}

// Structured implements log.Logger.
func (l *logger) Structured() log.Structured {
	return l
}

// WithName implements log.Logger.
func (l *logger) WithName(name string) log.Logger {
	return &logger{l.l.WithName(name), l.countByLevel}
}

// With implements log.LoggerWith
func (l *logger) With(fields ...log.Field) log.Logger {
	return &logger{log.With(l.l, fields...), l.countByLevel}
}

// With implements log.AddCallerSkip
func (l *logger) AddCallerSkip(skip int) log.Logger {
	return &logger{log.AddCallerSkip(l.l, skip), l.countByLevel}
}

////////////////////////////////////////////////////////////////////////////////

// Debug implements log.Logger.
func (l *logger) Debug(msg string, fields ...log.Field) {
	l.countByLevel[log.DebugLevel].Inc()
	l.l.Debug(msg, fields...)
}

// Debugf implements log.Logger.
func (l *logger) Debugf(format string, args ...interface{}) {
	l.countByLevel[log.DebugLevel].Inc()
	l.l.Debugf(format, args...)
}

// Error implements log.Logger.
func (l *logger) Error(msg string, fields ...log.Field) {
	l.countByLevel[log.ErrorLevel].Inc()
	l.l.Error(msg, fields...)
}

// Errorf implements log.Logger.
func (l *logger) Errorf(format string, args ...interface{}) {
	l.countByLevel[log.ErrorLevel].Inc()
	l.l.Errorf(format, args...)
}

// Fatal implements log.Logger.
func (l *logger) Fatal(msg string, fields ...log.Field) {
	l.countByLevel[log.FatalLevel].Inc()
	l.l.Fatal(msg, fields...)
}

// Fatalf implements log.Logger.
func (l *logger) Fatalf(format string, args ...interface{}) {
	l.countByLevel[log.FatalLevel].Inc()
	l.l.Fatalf(format, args...)
}

// Info implements log.Logger.
func (l *logger) Info(msg string, fields ...log.Field) {
	l.countByLevel[log.InfoLevel].Inc()
	l.l.Info(msg, fields...)
}

// Infof implements log.Logger.
func (l *logger) Infof(format string, args ...interface{}) {
	l.countByLevel[log.InfoLevel].Inc()
	l.l.Infof(format, args...)
}

// Trace implements log.Logger.
func (l *logger) Trace(msg string, fields ...log.Field) {
	l.countByLevel[log.TraceLevel].Inc()
	l.l.Trace(msg, fields...)
}

// Tracef implements log.Logger.
func (l *logger) Tracef(format string, args ...interface{}) {
	l.countByLevel[log.TraceLevel].Inc()
	l.l.Tracef(format, args...)
}

// Warn implements log.Logger.
func (l *logger) Warn(msg string, fields ...log.Field) {
	l.countByLevel[log.WarnLevel].Inc()
	l.l.Warn(msg, fields...)
}

// Warnf implements log.Logger.
func (l *logger) Warnf(format string, args ...interface{}) {
	l.countByLevel[log.WarnLevel].Inc()
	l.l.Warnf(format, args...)
}

////////////////////////////////////////////////////////////////////////////////
