package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	uzap "go.uber.org/zap"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/nop"
	"github.com/yandex/perforator/library/go/core/log/zap"
)

var (
	msg        = "msg"
	msgfmt     = "%s %s"
	msgfmtargs = []interface{}{"hello", "world"}
	key        = "key"
	value      = "value"
	withKey    = "withKey"
	withValue  = "withValue"
)

var loggersToTest = []struct {
	name    string
	factory func(level log.Level) (log.Logger, error)
}{
	{
		name: "Zap",
		factory: func(level log.Level) (log.Logger, error) {
			cfg := zap.JSONConfig(level)
			// Disable output
			cfg.OutputPaths = []string{}
			cfg.ErrorOutputPaths = []string{}
			return zap.New(cfg)
		},
	},
	{
		name: "ZapNop",
		factory: func(level log.Level) (log.Logger, error) {
			return &zap.Logger{
				L: uzap.NewNop(),
			}, nil
		},
	},
	{
		name: "Nop",
		factory: func(level log.Level) (log.Logger, error) {
			return &nop.Logger{}, nil
		},
	},
}

func TestLoggers(t *testing.T) {
	for _, loggerInput := range loggersToTest {
		for _, level := range log.Levels() {
			t.Run("Construct "+loggerInput.name+level.String(), func(t *testing.T) {
				logger, err := loggerInput.factory(level)
				require.NoError(t, err)
				require.NotNil(t, logger)

				lfmt := logger.Fmt()
				require.NotNil(t, lfmt)

				l := lfmt.Structured()
				require.NotNil(t, l)
				require.Equal(t, logger, l)
			})

			t.Run("With "+loggerInput.name+level.String(), func(t *testing.T) {
				logger, err := loggerInput.factory(level)
				require.NoError(t, err)
				require.NotNil(t, logger)

				withField := log.String(withKey, withValue)
				loggerWith := log.With(logger, withField)
				require.NotNil(t, loggerWith)
			})

			t.Run("AddCallerSkip "+loggerInput.name+level.String(), func(t *testing.T) {
				logger, err := loggerInput.factory(level)
				require.NoError(t, err)
				require.NotNil(t, logger)

				loggerCallerSkip := log.AddCallerSkip(logger, 1)
				require.NotNil(t, loggerCallerSkip)
			})

			// TODO: validate log output
			t.Run("Logger "+loggerInput.name+level.String(), func(t *testing.T) {
				logger, err := loggerInput.factory(level)
				require.NoError(t, err)
				require.NotNil(t, logger)

				logger.Trace(msg, log.String(key, value))
				logger.Debug(msg, log.String(key, value))
				logger.Info(msg, log.String(key, value))
				logger.Warn(msg, log.String(key, value))
				logger.Error(msg, log.String(key, value))
				// TODO: test fatal
			})

			// TODO: validate log output
			t.Run("LoggerFMT "+loggerInput.name+level.String(), func(t *testing.T) {
				logger, err := loggerInput.factory(level)
				require.NoError(t, err)
				require.NotNil(t, logger)

				lfmt := logger.Fmt()
				require.NotNil(t, lfmt)

				lfmt.Tracef(msgfmt, msgfmtargs...)
				lfmt.Debugf(msgfmt, msgfmtargs...)
				lfmt.Infof(msgfmt, msgfmtargs...)
				lfmt.Warnf(msgfmt, msgfmtargs...)
				lfmt.Errorf(msgfmt, msgfmtargs...)
				// TODO: test fatal
			})
		}
	}
}

func TestLazyCall(t *testing.T) {
	cfg := zap.JSONConfig(log.WarnLevel)
	// Disable output
	cfg.OutputPaths = []string{}
	cfg.ErrorOutputPaths = []string{}

	logger, err := zap.New(cfg)
	require.NoError(t, err)
	require.NotNil(t, logger)

	t.Run("will_not_call", func(t *testing.T) {
		var numCalls int
		logger.Info("I'm informing you", log.Lazy("problem", func() (any, error) {
			numCalls++
			return "something wicked this way comes", nil
		}))
		assert.Equal(t, 0, numCalls)
	})

	t.Run("will_call", func(t *testing.T) {
		var numCalls int
		logger.Error("I'm screaming for you", log.Lazy("problem", func() (any, error) {
			numCalls++
			return "something wicked this way comes", nil
		}))
		assert.Equal(t, 1, numCalls)
	})
}
