package tracing

type ExporterConfig struct {
	OTLP   *OTLPExporterConfig `yaml:"otlp"`
	Stderr *struct{}           `yaml:"stderr"`
	Nop    *struct{}           `yaml:"nop"`
}

type Config struct {
	Exporters []ExporterConfig `yaml:"exporters"`
}

const defaultOTLPEndpointEnv = "TRACING_OTLP_GRPC"

func NewDefaultConfig() *Config {
	return &Config{
		Exporters: []ExporterConfig{{
			OTLP: &OTLPExporterConfig{
				EndointEnv: defaultOTLPEndpointEnv,
			},
		}},
	}
}
