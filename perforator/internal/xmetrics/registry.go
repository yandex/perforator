package xmetrics

import (
	"context"
	"io"
	"net/http"

	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type Registry interface {
	metrics.Registry

	HTTPHandler(ctx context.Context, logger xlog.Logger) http.Handler
	StreamMetrics(ctx context.Context, w io.Writer) error
}
