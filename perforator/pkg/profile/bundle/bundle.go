package bundle

import (
	"errors"
	"sync"

	"github.com/yandex/perforator/perforator/pkg/cprofile"
)

type ProfileBundle struct {
	mu         sync.RWMutex
	pprofBody  []byte
	yaprofBody []byte
}

func NewPprofBundle(data []byte) *ProfileBundle {
	return &ProfileBundle{pprofBody: data}
}

func NewYaprofBundle(data []byte) *ProfileBundle {
	return &ProfileBundle{yaprofBody: data}
}

func NewBundle(pprof, yaprof []byte) *ProfileBundle {
	return &ProfileBundle{pprofBody: pprof, yaprofBody: yaprof}
}

func (b *ProfileBundle) GetOrConvertPprof() ([]byte, error) {
	b.mu.RLock()
	if len(b.pprofBody) > 0 {
		defer b.mu.RUnlock()
		return b.pprofBody, nil
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Double-check after acquiring write lock.
	if len(b.pprofBody) > 0 {
		return b.pprofBody, nil
	}
	if len(b.yaprofBody) == 0 {
		return nil, errors.New("profile bundle is empty: neither pprof nor yaprof data is available")
	}
	pprof, err := cprofile.YaprofToPProf(b.yaprofBody)
	if err != nil {
		return nil, err
	}
	b.pprofBody = pprof
	return pprof, nil
}

func (b *ProfileBundle) GetOrConvertYaprof() ([]byte, error) {
	b.mu.RLock()
	if len(b.yaprofBody) > 0 {
		defer b.mu.RUnlock()
		return b.yaprofBody, nil
	}
	b.mu.RUnlock()

	b.mu.Lock()
	defer b.mu.Unlock()

	// Double-check after acquiring write lock.
	if len(b.yaprofBody) > 0 {
		return b.yaprofBody, nil
	}
	if len(b.pprofBody) == 0 {
		return nil, errors.New("profile bundle is empty: neither pprof nor yaprof data is available")
	}
	yaprof, err := cprofile.PprofToYaprof(b.pprofBody)
	if err != nil {
		return nil, err
	}
	b.yaprofBody = yaprof
	return yaprof, nil
}

func (b *ProfileBundle) GetPprof() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.pprofBody
}

func (b *ProfileBundle) GetYaprof() []byte {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.yaprofBody
}
