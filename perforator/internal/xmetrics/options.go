package xmetrics

import "github.com/yandex/perforator/library/go/core/metrics/collect"

type Format int

const (
	FormatUnspecified Format = iota
	FormatBinary
	FormatText
)

type config struct {
	format     Format
	collectors []collect.Func
}

type Option func(*config)

func WithFormat(format Format) Option {
	return func(c *config) {
		c.format = format
	}
}

func WithAddCollectors(collectors ...collect.Func) Option {
	return func(c *config) {
		c.collectors = append(c.collectors, collectors...)
	}
}

func collectOptions(options ...Option) *config {
	conf := &config{}
	for _, opt := range options {
		opt(conf)
	}
	return conf
}
