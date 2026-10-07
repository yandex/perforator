package process

import (
	"context"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/dso"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xelf"
)

type Mapping interface {
	Path() string
	BinaryClass() dso.BinaryClass
	ID() uint64
	BaseAddress() uint64
	begin() uint64
	end() uint64
	buildInfo() *xelf.BuildInfo
}

type ProcessInfo interface {
	// Key identifies the process lifetime. ProcessStartTime is zero until known.
	Key() linux.ProcessKey
	// returned map may not be modified
	Env() map[string]string
	// Only fully synchronized BPF mappings are returned. The slice may not be modified.
	Mappings() []Mapping
}

type Listener interface {
	// Discovery and rescan are reported only after the process is installed in BPF.
	OnProcessDiscovery(ctx context.Context, info ProcessInfo)
	OnProcessRescan(ctx context.Context, info ProcessInfo)
	OnProcessDeath(ctx context.Context, key linux.ProcessKey)
}

// MetadataPublisher synchronously publishes sample metadata before BPF activation.
// It is separate from Listener, whose callbacks report a process ready to sample.
// The registry serializes publication and death per PID and keeps ProcessInfo
// stable for the duration of each callback. Callbacks must not reenter discovery.
// With a publisher, updated mappings must not become available for new sampling
// until publication and activation succeed, including on rescans. Failed updates
// may be retried within the same process lifetime. Samples already in flight or
// buffered in the perf ring are not drained by this contract.
type MetadataPublisher interface {
	PublishProcess(ctx context.Context, info ProcessInfo) error
	OnProcessDeath(ctx context.Context, key linux.ProcessKey)
}
