package tracing

import (
	"context"
	"errors"
	"fmt"
	"os"

	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/trace"
)

////////////////////////////////////////////////////////////////////////////////

type nopExporter struct{}

func (*nopExporter) ExportSpans(context.Context, []trace.ReadOnlySpan) error {
	return nil
}

func (*nopExporter) Shutdown(context.Context) error {
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func NewNopExporter() trace.SpanExporter {
	return &nopExporter{}
}

func NewStderrExporter(ctx context.Context) (trace.SpanExporter, error) {
	return stdouttrace.New(
		stdouttrace.WithPrettyPrint(),
		stdouttrace.WithWriter(os.Stderr),
	)
}

////////////////////////////////////////////////////////////////////////////////

func NewMultiExporter(exporters ...trace.SpanExporter) trace.SpanExporter {
	return &multiExporter{exporters}
}

type multiExporter struct {
	exporters []trace.SpanExporter
}

// ExportSpans implements trace.SpanExporter.
func (e *multiExporter) ExportSpans(ctx context.Context, spans []trace.ReadOnlySpan) error {
	return e.do(func(exp trace.SpanExporter) error {
		return exp.ExportSpans(ctx, spans)
	})
}

// Shutdown implements trace.SpanExporter.
func (e *multiExporter) Shutdown(ctx context.Context) error {
	return e.do(func(exp trace.SpanExporter) error {
		return exp.Shutdown(ctx)
	})
}

func (e *multiExporter) do(callback func(trace.SpanExporter) error) error {
	errs := make([]error, 0)
	for _, exporter := range e.exporters {
		err := callback(exporter)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

////////////////////////////////////////////////////////////////////////////////

func NewExporter(ctx context.Context, config *Config) (trace.SpanExporter, error) {
	exporters := []trace.SpanExporter{}
	for _, exporterConfig := range config.Exporters {
		var exporter trace.SpanExporter
		var err error

		switch {
		case exporterConfig.Nop != nil:
			exporter = NewNopExporter()
		case exporterConfig.Stderr != nil:
			exporter, err = NewStderrExporter(ctx)
		case exporterConfig.OTLP != nil:
			exporter, err = NewOTLPExporter(ctx, *exporterConfig.OTLP)
		default:
			err = fmt.Errorf("malformed trace exporter config")
		}

		if err != nil {
			return nil, err
		}

		exporters = append(exporters, exporter)
	}

	switch len(exporters) {
	case 0:
		return NewNopExporter(), nil
	case 1:
		return exporters[0], nil
	default:
		return NewMultiExporter(exporters...), nil
	}
}

////////////////////////////////////////////////////////////////////////////////
