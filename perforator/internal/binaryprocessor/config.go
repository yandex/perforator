package binaryprocessor

import (
	"os"

	"gopkg.in/yaml.v3"

	proxy "github.com/yandex/perforator/perforator/internal/symbolizer/proxy/server"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/tracing"
)

type Config struct {
	StorageConfig       bundle.Config              `yaml:"storage"`
	BinaryProvider      proxy.BinaryProviderConfig `yaml:"binary_provider"`
	SymbolizationConfig proxy.SymbolizationConfig  `yaml:"symbolization"`
	Tracing             *tracing.Config            `yaml:"tracing"`
}

func ParseConfig(path string) (conf *Config, err error) {
	var file *os.File
	file, err = os.Open(path)
	if err != nil {
		return nil, err
	}

	conf = &Config{}
	if err = yaml.NewDecoder(file).Decode(conf); err != nil {
		return nil, err
	}

	if conf.Tracing == nil {
		conf.Tracing = tracing.NewDefaultConfig()
	}
	return conf, nil
}
