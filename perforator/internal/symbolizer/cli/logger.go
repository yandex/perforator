package cli

import (
	"os"

	"github.com/mattn/go-isatty"
	uberzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/log/zap"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func NewLogger(level log.Level) (xlog.Logger, error) {
	config := uberzap.NewDevelopmentConfig()
	config.Level = uberzap.NewAtomicLevelAt(zap.ZapifyLevel(level))

	if isatty.IsTerminal(os.Stderr.Fd()) {
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	}

	config.EncoderConfig.ConsoleSeparator = " "
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(`15:04:05.000`)
	config.DisableStacktrace = true
	return xlog.TryNew(zap.New(config))
}
