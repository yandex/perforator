package pion

import (
	"github.com/pion/logging"

	"github.com/yandex/perforator/library/go/core/log"
)

type LoggerFactory struct {
	StandardLogger log.Logger
}

func (l LoggerFactory) NewLogger(scope string) logging.LeveledLogger {
	return LoggerAdapter{
		standardLogger: l.StandardLogger,
		scope:          scope,
	}
}

type LoggerAdapter struct {
	standardLogger log.Logger
	scope          string
}

func (a LoggerAdapter) Trace(msg string) {
	log.AddCallerSkip(a.standardLogger, 1).Trace(a.addScope(msg))
}

func (a LoggerAdapter) Tracef(format string, args ...interface{}) {
	log.AddCallerSkip(a.standardLogger, 1).Tracef(a.addScope(format), args...)
}

func (a LoggerAdapter) Debug(msg string) {
	log.AddCallerSkip(a.standardLogger, 1).Debug(a.addScope(msg))
}

func (a LoggerAdapter) Debugf(format string, args ...interface{}) {
	log.AddCallerSkip(a.standardLogger, 1).Debugf(a.addScope(format), args...)
}

func (a LoggerAdapter) Info(msg string) {
	log.AddCallerSkip(a.standardLogger, 1).Info(a.addScope(msg))
}

func (a LoggerAdapter) Infof(format string, args ...interface{}) {
	log.AddCallerSkip(a.standardLogger, 1).Infof(a.addScope(format), args...)
}

func (a LoggerAdapter) Warn(msg string) {
	log.AddCallerSkip(a.standardLogger, 1).Warn(a.addScope(msg))
}

func (a LoggerAdapter) Warnf(format string, args ...interface{}) {
	log.AddCallerSkip(a.standardLogger, 1).Warnf(a.addScope(format), args...)
}

func (a LoggerAdapter) Error(msg string) {
	log.AddCallerSkip(a.standardLogger, 1).Error(a.addScope(msg))
}

func (a LoggerAdapter) Errorf(format string, args ...interface{}) {
	log.AddCallerSkip(a.standardLogger, 1).Errorf(a.addScope(format), args...)
}

func (a LoggerAdapter) addScope(s string) string {
	return a.scope + ": " + s
}
