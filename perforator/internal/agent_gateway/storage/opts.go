package storage

import (
	"os"
	"time"
)

type options struct {
	clusterName            string
	samplingModulo         uint64
	samplingModuloByEvent  map[string]uint64
	maxBuildIDCacheEntries uint64
	pushProfileTimeout     time.Duration
	pushBinaryWriteAbility bool
}

func defaultOpts() *options {
	return &options{
		clusterName:            os.Getenv("DEPLOY_NODE_DC"),
		samplingModulo:         1,
		maxBuildIDCacheEntries: 14000000,
		pushProfileTimeout:     10 * time.Second,
		pushBinaryWriteAbility: true,
	}
}

type Option func(*options)

func WithClusterName(clusterName string) Option {
	return func(o *options) {
		o.clusterName = clusterName
	}
}

func WithSamplingModulo(samplingModulo uint64) Option {
	return func(o *options) {
		o.samplingModulo = samplingModulo
	}
}

func WithSamplingModuloByEvent(samplingModuloByEvent map[string]uint64) Option {
	return func(o *options) {
		o.samplingModuloByEvent = samplingModuloByEvent
	}
}
func WithMaxBuildIDCacheEntries(maxBuildIDCacheEntries uint64) Option {
	return func(o *options) {
		o.maxBuildIDCacheEntries = maxBuildIDCacheEntries
	}
}

func WithPushProfileTimeout(pushProfileTimeout time.Duration) Option {
	return func(o *options) {
		o.pushProfileTimeout = pushProfileTimeout
	}
}

func WithPushBinaryWriteAbility(pushBinaryWriteAbility bool) Option {
	return func(o *options) {
		o.pushBinaryWriteAbility = pushBinaryWriteAbility
	}
}
