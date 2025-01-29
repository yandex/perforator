package tracing

import (
	"context"
	"errors"

	"github.com/go-logr/logr"
	"github.com/opentracing/opentracing-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelBridge "go.opentelemetry.io/otel/bridge/opentracing"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/yandex/perforator/library/go/core/buildinfo"
	"github.com/yandex/perforator/library/go/core/log"
)

func Initialize(
	ctx context.Context,
	log log.Logger,
	exporter sdktrace.SpanExporter,
	projectName string,
	serviceName string,
) (
	shutdown func(context.Context) error,
	tracer trace.TracerProvider,
	err error,
) {
	shutdownFuncs := []func(context.Context) error{
		exporter.Shutdown,
	}

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	////////////////////////////////////////////////////////////////////////////////

	setOpenTelemetryLogger(log)

	////////////////////////////////////////////////////////////////////////////////

	// Set up resource.
	res, err := resource.Merge(resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(buildinfo.Info.ArcadiaSourceRevision),
			attribute.String("project", projectName),
		))
	if err != nil {
		handleErr(err)
		return
	}

	////////////////////////////////////////////////////////////////////////////////

	// Set up propagator.
	prop := propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
	otel.SetTextMapPropagator(prop)

	////////////////////////////////////////////////////////////////////////////////

	// Set up trace provider.
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	shutdownFuncs = append(shutdownFuncs, tp.Shutdown)

	////////////////////////////////////////////////////////////////////////////////

	// Setup compatibility layer with the opentracing.
	bridge := otelBridge.NewBridgeTracer()
	tracer = otelBridge.NewTracerProvider(bridge, tp)

	// Set tracers as the default trace providers.
	otel.SetTracerProvider(tracer)
	opentracing.SetGlobalTracer(bridge)

	////////////////////////////////////////////////////////////////////////////////

	return
}

func setOpenTelemetryLogger(l log.Logger) {
	logr := logr.New(&logrZapSink{l, 0})
	otel.SetLogger(logr)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logr.Error(err, "opentelemetry error")
	}))
}
