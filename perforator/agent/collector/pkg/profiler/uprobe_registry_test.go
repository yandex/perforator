package profiler

import (
	"errors"
	"testing"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/collector/pkg/uprobe"
)

type failingCloseUprobe struct {
	info       *uprobe.UprobeInfo
	closeErr   error
	closeCalls int
}

func (u *failingCloseUprobe) Attach(*ebpf.Program) error { return nil }
func (u *failingCloseUprobe) Info() *uprobe.UprobeInfo {
	info := *u.info // Like the real uprobe, Info is invalid after Close.
	return &info
}
func (u *failingCloseUprobe) Close() error {
	u.closeCalls++
	u.info = nil // The real uprobe clears its metadata even on failure.
	return u.closeErr
}

func TestUprobeCloseAfterFailedDetach(t *testing.T) {
	closeErr := errors.New("link close failed")
	info := &uprobe.UprobeInfo{BinaryInfo: uprobe.BinaryInfo{BuildID: "test", Offset: 42}}
	probe := &failingCloseUprobe{info: info, closeErr: closeErr}
	registry := newUprobeRegistry(nil)
	wrapped := &uprobeWrapper{uprobe: probe, attached: true, registry: registry}
	wrapped.registryID = registry.register(wrapped)
	registry.Resolver.Add(info)

	require.ErrorIs(t, wrapped.Detach(), closeErr)
	// CPO Stop calls Close after Detach, including when Detach failed.
	require.NotPanics(t, func() { require.NoError(t, wrapped.Close()) })
	require.False(t, wrapped.attached)
	require.Empty(t, registry.Resolve(info.BinaryInfo))
	require.Empty(t, registry.uprobes)
	require.Equal(t, 1, probe.closeCalls)
}
