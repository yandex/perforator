package storage

import "github.com/yandex/perforator/perforator/pkg/storage/microscope/filter"

type ServiceConfig struct {
	MicroscopePullerConfig *filter.Config `yaml:"microscope_puller"`
}
