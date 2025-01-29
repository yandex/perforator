package xlog

import (
	"context"

	"go.opentelemetry.io/otel/trace"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/ctxlog"
	"github.com/yandex/perforator/library/go/core/log/nop"
)

////////////////////////////////////////////////////////////////////////////////

type Logger interface {
	With(fields ...log.Field) Logger
	WithName(name string) Logger
	WithCallerSkip(level int) Logger

	WithContext(ctx context.Context) log.Logger
	Logger() log.Logger
	Fmt() log.Fmt

	Trace(ctx context.Context, msg string, fields ...log.Field)
	Debug(ctx context.Context, msg string, fields ...log.Field)
	Info(ctx context.Context, msg string, fields ...log.Field)
	Warn(ctx context.Context, msg string, fields ...log.Field)
	Error(ctx context.Context, msg string, fields ...log.Field)
	Fatal(ctx context.Context, msg string, fields ...log.Field)
}

var WrapContext = ctxlog.WithFields

////////////////////////////////////////////////////////////////////////////////

type logger struct {
	log log.Logger
}

var _ Logger = (*logger)(nil)

////////////////////////////////////////////////////////////////////////////////

func New(log log.Logger) Logger {
	return &logger{log}
}

func NewNop() Logger {
	return &logger{&nop.Logger{}}
}

func TryNew(log log.Logger, err error) (Logger, error) {
	if err != nil {
		return nil, err
	}
	return New(log), nil
}

func (l *logger) Logger() log.Logger {
	return l.log
}

func (l *logger) Fmt() log.Fmt {
	return l.log.Fmt()
}

////////////////////////////////////////////////////////////////////////////////

func (l *logger) With(fields ...log.Field) Logger {
	return &logger{log.With(l.log, fields...)}
}

func (l *logger) WithName(name string) Logger {
	return &logger{l.log.WithName(name)}
}

func (l *logger) WithContext(ctx context.Context) log.Logger {
	return &boundLogger{l, ctx}
}

func (l *logger) WithCallerSkip(level int) Logger {
	return &logger{log.AddCallerSkip(l.log, level)}
}

////////////////////////////////////////////////////////////////////////////////

func (l *logger) withCallerSkip(level int) log.Logger {
	return log.AddCallerSkip(l.log, level)
}

func (l *logger) Trace(ctx context.Context, msg string, fields ...log.Field) {
	ctxlog.Trace(ctx, l.withCallerSkip(1), msg, addTraceFields(ctx, fields)...)
}

func (l *logger) Debug(ctx context.Context, msg string, fields ...log.Field) {
	ctxlog.Debug(ctx, l.withCallerSkip(1), msg, addTraceFields(ctx, fields)...)
}

func (l *logger) Info(ctx context.Context, msg string, fields ...log.Field) {
	ctxlog.Info(ctx, l.withCallerSkip(1), msg, addTraceFields(ctx, fields)...)
}

func (l *logger) Warn(ctx context.Context, msg string, fields ...log.Field) {
	ctxlog.Warn(ctx, l.withCallerSkip(1), msg, addTraceFields(ctx, fields)...)
}

func (l *logger) Error(ctx context.Context, msg string, fields ...log.Field) {
	ctxlog.Error(ctx, l.withCallerSkip(1), msg, addTraceFields(ctx, fields)...)
}

func (l *logger) Fatal(ctx context.Context, msg string, fields ...log.Field) {
	ctxlog.Fatal(ctx, l.withCallerSkip(1), msg, addTraceFields(ctx, fields)...)
}

func addTraceFields(ctx context.Context, fields []log.Field) []log.Field {
	span := trace.SpanContextFromContext(ctx)
	if span.HasTraceID() {
		fields = append(fields, log.String("trace.id", span.TraceID().String()))
	}
	if span.HasSpanID() {
		fields = append(fields, log.String("span.id", span.SpanID().String()))
	}
	return fields
}

////////////////////////////////////////////////////////////////////////////////
