package profiler

import (
	"errors"
	"fmt"
	"sync"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/machine"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/uprobe"
)

// Uprobe is a Profiler's user interface for a single uprobe.
// Not guaranteed to be thread-safe.
type Uprobe interface {
	// Attach attaches the uprobe to the bpf program which collects samples. Must be called on detached uprobe.
	// Attach is safe to call multiple times during Uprobe lifetime,
	Attach() error

	// Detach detaches the uprobe from the bpf program. Must be called on attached uprobe.
	// Detach is safe to call multiple times during Uprobe lifetime,
	Detach() error

	// Close detaches the uprobe from the bpf program and releases the resources.
	// It is safe to close both attached and detached uprobe.
	Close() error
}

// UprobeManager is a Profiler's user interface for
// dynamic uprobe creation and deletion.
type UprobeManager interface {
	Create(config uprobe.Config) Uprobe
}

// uprobeRegistry is a container which stores all the created uprobes.
// It provides these functions:
// 1. Create uprobe
// 2. Resolve profiler samples into specific uprobes
// 3. Reattach uprobes on bpf program reload (e.g. after SetDebugMode)
//
// uprobeRegistry assumes all uprobes are linked to the same single bpf program.
type uprobeRegistry struct {
	sync.Mutex
	id      uint64
	uprobes map[uint64]*uprobeWrapper
	*uprobe.Resolver
	bpf *machine.BPF
}

func newUprobeRegistry(bpf *machine.BPF) *uprobeRegistry {
	return &uprobeRegistry{
		id:       0,
		uprobes:  make(map[uint64]*uprobeWrapper),
		Resolver: uprobe.NewResolver(),
		bpf:      bpf,
	}
}

func (r *uprobeRegistry) detachAll() error {
	r.Lock()
	defer r.Unlock()

	var errs []error
	for _, uprobe := range r.uprobes {
		err := uprobe.Detach()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (r *uprobeRegistry) attachAll() error {
	r.Lock()
	defer r.Unlock()

	var errs []error
	for _, uprobe := range r.uprobes {
		err := uprobe.Attach()
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (r *uprobeRegistry) register(uprobe *uprobeWrapper) (id uint64) {
	r.Lock()
	defer r.Unlock()

	id = r.id
	r.id++
	r.uprobes[id] = uprobe

	return
}

func (r *uprobeRegistry) unregister(id uint64) {
	r.Lock()
	defer r.Unlock()

	_, ok := r.uprobes[id]
	if !ok {
		panic("unregistering uprobe which is not in a registry")
	}

	delete(r.uprobes, id)
}

type uprobeWrapper struct {
	uprobe     uprobe.Uprobe
	attached   bool
	registryID uint64
	registry   *uprobeRegistry
}

func (u *uprobeWrapper) Attach() error {
	program := u.registry.bpf.GenericUprobeProgram()
	if program == nil {
		// sanity check
		return errors.New("generic uprobe program is not loaded")
	}

	err := u.uprobe.Attach(program)
	if err != nil {
		return err
	}

	u.attached = true
	u.registry.Resolver.Add(u.uprobe.Info())

	return nil
}

func (u *uprobeWrapper) Detach() error {
	if !u.attached {
		return nil
	}

	binaryInfo := u.uprobe.Info().BinaryInfo
	err := u.uprobe.Close()
	if err != nil {
		return err
	}

	u.attached = false
	u.registry.Resolver.Remove(binaryInfo)
	return nil
}

func (u *uprobeWrapper) Close() error {
	err := u.Detach()
	if err != nil {
		return fmt.Errorf("failed to detach uprobe: %w", err)
	}

	u.registry.unregister(u.registryID)
	return nil
}

func (r *uprobeRegistry) Create(config uprobe.Config) Uprobe {
	uprobe := &uprobeWrapper{
		uprobe:   uprobe.NewUprobe(config),
		registry: r,
	}

	uprobe.registryID = r.register(uprobe)
	return uprobe
}
