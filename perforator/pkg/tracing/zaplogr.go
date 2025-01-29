package tracing

import (
	"github.com/go-logr/logr"

	"github.com/yandex/perforator/library/go/core/log"
)

type logrZapSink struct {
	l     log.Logger
	level int
}

var _ logr.LogSink = (*logrZapSink)(nil)

func fieldify(kv ...interface{}) []log.Field {
	if len(kv)%2 != 0 {
		panic("keys and values are not interleaved")
	}
	fields := make([]log.Field, 0, len(kv)/2)
	for i := 0; i < len(kv); i += 2 {
		key, ok := kv[i].(string)
		if !ok {
			panic("key is not a string")
		}
		value := kv[i+1]
		fields = append(fields, log.Any(key, value))
	}
	return fields
}

// Enabled implements logr.LogSink
func (l *logrZapSink) Enabled(level int) bool {
	return level <= l.level
}

// Error implements logr.LogSink
func (l *logrZapSink) Error(err error, msg string, kv ...interface{}) {
	l.l.Error(msg, append(fieldify(kv...), log.Error(err))...)
}

// Info implements logr.LogSink
func (l *logrZapSink) Info(level int, msg string, kv ...interface{}) {
	if level > l.level {
		return
	}
	l.l.Info(msg, fieldify(kv...)...)
}

// Init implements logr.LogSink
func (l *logrZapSink) Init(info logr.RuntimeInfo) {
	l.l = log.AddCallerSkip(l.l, info.CallDepth)
}

// WithName implements logr.LogSink
func (l *logrZapSink) WithName(name string) logr.LogSink {
	return &logrZapSink{l.l.WithName(name), l.level}
}

// WithValues implements logr.LogSink
func (l *logrZapSink) WithValues(kv ...interface{}) logr.LogSink {
	return &logrZapSink{log.With(l.l, fieldify(kv...)...), l.level}
}
