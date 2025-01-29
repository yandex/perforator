package tracing

import (
	"context"
	"os"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/trace"
)

////////////////////////////////////////////////////////////////////////////////

type OTLPExporterConfig struct {
	EndointEnv string `yaml:"endpoint_env"`
	Endoint    string `yaml:"endpoint"`
	Secure     bool   `yaml:"secure"`
}

func NewOTLPExporter(ctx context.Context, conf OTLPExporterConfig) (trace.SpanExporter, error) {
	endpoint := conf.Endoint
	if endpoint == "" && conf.EndointEnv != "" {
		endpoint = os.Getenv(conf.EndointEnv)
	}

	if endpoint == "" {
		return NewNopExporter(), nil
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(endpoint),
	}

	if !conf.Secure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	return otlptracegrpc.New(ctx, opts...)

}

////////////////////////////////////////////////////////////////////////////////
