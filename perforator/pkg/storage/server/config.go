package server

import (
	"os"

	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope/filter"
)

type TvmAuth struct {
	ID            uint32 `yaml:"id"`
	SecretEnvName string `yaml:"secret_env"`
}

type TLSConfig struct {
	CertificateFile string `yaml:"certificate_file"`
	KeyFile         string `yaml:"key_file"`
}

type Config struct {
	Port                   uint32         `yaml:"port"`
	MetricsPort            uint32         `yaml:"metrics_port"`
	StorageConfig          bundle.Config  `yaml:"storage"`
	TvmAuth                *TvmAuth       `yaml:"tvm"`
	TLSConfig              TLSConfig      `yaml:"tls"`
	MicroscopePullerConfig *filter.Config `yaml:"microscope_puller"`
}

func ParseConfig(path string, strict bool) (conf *Config, err error) {
	// TODO(PERFORATOR-480): always be strict
	var file *os.File
	file, err = os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	conf = &Config{}
	dec := yaml.NewDecoder(file)
	dec.KnownFields(strict)
	err = dec.Decode(conf)
	return
}
