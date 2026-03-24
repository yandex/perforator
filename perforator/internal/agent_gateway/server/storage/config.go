package storage

import (
	kafka "github.com/yandex/perforator/perforator/pkg/kafka/producer"
	"github.com/yandex/perforator/perforator/pkg/profile_event/async_publisher"
	"github.com/yandex/perforator/perforator/pkg/storage/microscope/filter"
)

type ProfileSignalEventsConfig struct {
	async_publisher.Config
	SamplingRate   float64       `yaml:"sampling_rate"`
	AllowedSignals []string      `yaml:"allowed_signals"` // e.g, SIGSEGV, SIGQUIT
	Kafka          *kafka.Config `yaml:"kafka"`
}

type ServiceConfig struct {
	MicroscopePullerConfig *filter.Config             `yaml:"microscope_puller"`
	ProfileSignalEvents    *ProfileSignalEventsConfig `yaml:"profile_signal_events"`
}
