package agent

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/library/go/core/metrics/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/profile"
	"github.com/yandex/perforator/perforator/internal/linguist/models"
	"github.com/yandex/perforator/perforator/internal/linguist/python/agent/linetable"
	"github.com/yandex/perforator/perforator/internal/linguist/symbolizer"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

type nopSymbolSource struct{}

func (nopSymbolSource) Symbolize(models.Language, linux.ProcessKey, *unwinder.SymbolKey) (*symbolizer.Symbol, bool) {
	return nil, false
}

func TestNewRegistry_WithoutLineInfo(t *testing.T) {
	reg, err := newRegistry(xlog.NewNop(), nopSymbolSource{}, SymbolizerConfig{}, nop.Registry{})
	require.NoError(t, err)
	t.Cleanup(reg.Stop)
	require.NotNil(t, reg)
	require.Nil(t, reg.offsets)

	reg.ProcessStack(profile.NewBuilder().AddSampleType("cpu", "cycles").Add(linux.ProcessKey{Pid: 1}).AddValue(1), &unwinder.PythonStack{}, linux.ProcessKey{Pid: testPid})

	reg.OnProcessDeath(context.Background(), linux.ProcessKey{Pid: testPid})
}

func TestNewRegistry_WithLineInfo(t *testing.T) {
	reg, err := newRegistry(xlog.NewNop(), nopSymbolSource{}, SymbolizerConfig{}, nop.Registry{}, WithLineInfo())
	require.NoError(t, err)
	t.Cleanup(reg.Stop)
	require.NotNil(t, reg)
	require.NotNil(t, reg.offsets)

	reg.ProcessStack(profile.NewBuilder().AddSampleType("cpu", "cycles").Add(linux.ProcessKey{Pid: 1}).AddValue(1), &unwinder.PythonStack{}, linux.ProcessKey{Pid: testPid})

	reg.OnProcessDeath(context.Background(), linux.ProcessKey{Pid: testPid})
}

func TestRegistry_StopIdempotent(t *testing.T) {
	reg, err := newRegistry(xlog.NewNop(), nopSymbolSource{}, SymbolizerConfig{}, nop.Registry{}, WithLineInfo())
	require.NoError(t, err)

	reg.Stop()
	reg.Stop()
}

func TestRegistry_ProcessDeathInvalidatesLinetableCache(t *testing.T) {
	reg, err := newRegistry(xlog.NewNop(), nopSymbolSource{}, SymbolizerConfig{}, nop.Registry{}, WithLineInfo())
	require.NoError(t, err)
	t.Cleanup(reg.Stop)

	key := linetable.CacheKey{Process: linux.ProcessKey{Pid: testPid}}
	reg.symbolizer.cache.AddTombstone(key)
	_, ok := reg.symbolizer.cache.Get(key)
	require.True(t, ok)

	reg.OnProcessDeath(context.Background(), linux.ProcessKey{Pid: testPid})
	_, ok = reg.symbolizer.cache.Get(key)
	require.False(t, ok)
}
