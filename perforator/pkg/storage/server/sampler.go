package server

import (
	"sync/atomic"
)

type Sampler interface {
	Sample() bool
}

type ModuloSampler struct {
	counter atomic.Uint64
	modulo  uint64
}

func (s *ModuloSampler) Sample() bool {
	newValue := s.counter.Add(1)
	return newValue%s.modulo == 0
}

func NewModuloSampler(modulo uint64) Sampler {
	if modulo == 0 {
		modulo = 1
	}

	return &ModuloSampler{
		modulo: modulo,
	}
}
