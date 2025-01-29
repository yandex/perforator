package app

import (
	"os"

	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/perforator/internal/asyncfilecache"
	"github.com/yandex/perforator/perforator/pkg/storage/bundle"
)

type BinaryProviderConfig struct {
	FileCache                *asyncfilecache.Config `yaml:"file_cache"`
	MaxSimultaneousDownloads uint32                 `yaml:"max_simultaneous_downloads"`
}

type Config struct {
	StorageConfig  bundle.Config        `yaml:"storage"`
	BinaryProvider BinaryProviderConfig `yaml:"binary_provider"`

	GsymS3Bucket string `yaml:"gsym_s3_bucket"`
}

func ParseConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	conf := &Config{}
	err = yaml.NewDecoder(file).Decode(conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
