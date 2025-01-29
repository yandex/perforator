package xlog

import (
	"context"

	"github.com/yandex/perforator/library/go/core/log"
)

////////////////////////////////////////////////////////////////////////////////

type boundLogger struct {
	l   Logger
	ctx context.Context
}

var _ log.Logger = (*boundLogger)(nil)
var _ log.LoggerWith = (*boundLogger)(nil)
var _ log.LoggerAddCallerSkip = (*boundLogger)(nil)

////////////////////////////////////////////////////////////////////////////////

// Fmt implements log.Logger.
func (b *boundLogger) Fmt() log.Fmt {
	return b
}

// Structured implements log.Logger.
func (b *boundLogger) Structured() log.Structured {
	return b
}

// WithName implements log.Logger.
func (b *boundLogger) WithName(name string) log.Logger {
	return &boundLogger{b.l.WithName(name), b.ctx}
}

// WithName implements log.Fmt.
func (b *boundLogger) Logger() log.Logger {
	return b
}

////////////////////////////////////////////////////////////////////////////////

// Trace implements log.Logger.
func (b *boundLogger) Trace(msg string, fields ...log.Field) {
	b.l.WithCallerSkip(1).Trace(b.ctx, msg, fields...)
}

// Tracef implements log.Logger.
func (b *boundLogger) Tracef(format string, args ...any) {
	b.l.WithCallerSkip(1).Fmt().Tracef(format, args...)
}

// Debug implements log.Logger.
func (b *boundLogger) Debug(msg string, fields ...log.Field) {
	b.l.WithCallerSkip(1).Debug(b.ctx, msg, fields...)
}

// Debugf implements log.Logger.
func (b *boundLogger) Debugf(format string, args ...any) {
	b.l.WithCallerSkip(1).Fmt().Debugf(format, args...)
}

// Info implements log.Logger.
func (b *boundLogger) Info(msg string, fields ...log.Field) {
	b.l.WithCallerSkip(1).Info(b.ctx, msg, fields...)
}

// Infof implements log.Logger.
func (b *boundLogger) Infof(format string, args ...any) {
	b.l.WithCallerSkip(1).Fmt().Infof(format, args...)
}

// Warn implements log.Logger.
func (b *boundLogger) Warn(msg string, fields ...log.Field) {
	b.l.WithCallerSkip(1).Warn(b.ctx, msg, fields...)
}

// Warnf implements log.Logger.
func (b *boundLogger) Warnf(format string, args ...any) {
	b.l.WithCallerSkip(1).Fmt().Warnf(format, args...)
}

// Error implements log.Logger.
func (b *boundLogger) Error(msg string, fields ...log.Field) {
	b.l.WithCallerSkip(1).Error(b.ctx, msg, fields...)
}

// Errorf implements log.Logger.
func (b *boundLogger) Errorf(format string, args ...any) {
	b.l.WithCallerSkip(1).Fmt().Errorf(format, args...)
}

// Fatal implements log.Logger.
func (b *boundLogger) Fatal(msg string, fields ...log.Field) {
	b.l.WithCallerSkip(1).Fatal(b.ctx, msg, fields...)
}

// Fatalf implements log.Logger.
func (b *boundLogger) Fatalf(format string, args ...any) {
	b.l.WithCallerSkip(1).Fmt().Fatalf(format, args...)
}

////////////////////////////////////////////////////////////////////////////////

// AddCallerSkip implements log.LoggerAddCallerSkip.
func (b *boundLogger) AddCallerSkip(skip int) log.Logger {
	return &boundLogger{l: b.l.WithCallerSkip(skip), ctx: b.ctx}
}

// With implements log.LoggerWith.
func (b *boundLogger) With(fields ...log.Field) log.Logger {
	return &boundLogger{l: b.l.With(fields...), ctx: b.ctx}
}

////////////////////////////////////////////////////////////////////////////////
