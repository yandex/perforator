package grpcmetrics

import (
	"context"
	"maps"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/yandex/perforator/library/go/core/metrics"
)

////////////////////////////////////////////////////////////////////////////////

type MetricsInterceptor struct {
	requestCount     metrics.CounterVec
	responseCount    metrics.CounterVec
	responseLatency  metrics.TimerVec
	inflightRequests metrics.IntGaugeVec
}

func NewMetricsInterceptor(registry metrics.Registry) *MetricsInterceptor {
	registry = registry.WithPrefix("grpc")

	latencyBuckets := metrics.MakeExponentialDurationBuckets(time.Millisecond, 1.3, 50)

	return &MetricsInterceptor{
		requestCount:     registry.CounterVec("request.count", []string{"method"}),
		responseCount:    registry.CounterVec("response.count", []string{"method", "status"}),
		responseLatency:  registry.DurationHistogramVec("latency.seconds", latencyBuckets, []string{"method", "status"}),
		inflightRequests: registry.IntGaugeVec("inflight.count", []string{"method"}),
	}
}

type requestTracker struct {
	parent *MetricsInterceptor
	start  time.Time
	tags   map[string]string
}

func (l *MetricsInterceptor) trackRequest(method string) *requestTracker {
	tags := make(map[string]string, 2)
	tags["method"] = method

	t := &requestTracker{
		parent: l,
		start:  time.Now(),
		tags:   tags,
	}
	t.begin()
	return t
}

func (t *requestTracker) begin() {
	t.parent.requestCount.With(t.tags).Inc()
	t.parent.inflightRequests.With(t.tags).Add(1)
}

func (t *requestTracker) done(err error) {
	t.parent.inflightRequests.With(t.tags).Add(-1)

	tags := maps.Clone(t.tags)
	tags["status"] = status.Code(err).String()
	t.parent.responseCount.With(tags).Add(1)
	t.parent.responseLatency.With(tags).RecordDuration(time.Since(t.start))
}

func (l *MetricsInterceptor) UnaryServer() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		tracker := l.trackRequest(info.FullMethod)
		defer func() {
			tracker.done(err)
		}()
		return handler(ctx, req)
	}
}

func (l *MetricsInterceptor) StreamServer() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) (err error) {
		tracker := l.trackRequest(info.FullMethod)
		defer func() {
			tracker.done(err)
		}()
		return handler(srv, ss)
	}
}
