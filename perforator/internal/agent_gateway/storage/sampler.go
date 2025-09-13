package storage

import (
	"sync/atomic"
)

type moduloSampler struct {
	counter atomic.Uint64
	modulo  uint64
}

func (s *moduloSampler) Sample() bool {
	newValue := s.counter.Add(1)
	return newValue%s.modulo == 0
}

func newModuloSampler(modulo uint64) *moduloSampler {
	if modulo == 0 {
		modulo = 1
	}

	return &moduloSampler{
		modulo: modulo,
	}
}
