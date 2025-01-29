package xmetrics

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics/collect"
	"github.com/yandex/perforator/library/go/core/metrics/collect/policy/inflight"
	"github.com/yandex/perforator/library/go/core/metrics/prometheus"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func GetCollectFuncs() []collect.Func {
	return []collect.Func{}
}

type prometheusHTTPHandler struct {
	logCtx   context.Context
	logger   xlog.Logger
	registry *prometheus.Registry
}

func (h *prometheusHTTPHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	_, err := h.registry.Stream(ctx, rw)
	if err != nil {
		h.logger.Warn(
			h.logCtx,
			"Failed to serve metrics",
			log.Error(err),
		)
	}
}

type prometheusRegistry struct {
	*prometheus.Registry
}

func (r prometheusRegistry) HTTPHandler(ctx context.Context, logger xlog.Logger) http.Handler {
	return &prometheusHTTPHandler{
		logCtx:   ctx,
		logger:   logger,
		registry: r.Registry,
	}
}

func (r prometheusRegistry) StreamMetrics(ctx context.Context, w io.Writer) error {
	_, err := r.Stream(ctx, w)
	return err
}

func NewRegistry(options ...Option) Registry {
	conf := collectOptions(options...)
	regOpts := prometheus.NewRegistryOpts()
	if len(conf.collectors) > 0 {
		regOpts.AddCollectors(context.Background(), inflight.NewCollectorPolicy(), conf.collectors...)
	}

	if conf.format != FormatBinary {
		regOpts.SetStreamFormat(prometheus.StreamText)
	}

	regOpts.SetNameSanitizer(sanitizePrometheusMetricName)

	registry := prometheus.NewRegistry(regOpts)
	return prometheusRegistry{registry}
}

// See https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
var prometheusMetricSanitizer = strings.NewReplacer(
	".", "_",
	"-", "_",
)

func sanitizePrometheusMetricName(name string) string {
	return prometheusMetricSanitizer.Replace(name)
}
