package podtracker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/yandex/perforator/library/go/core/log/nop"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/deploysystem"
)

type snapshotSourceFunc func(context.Context) (deploysystem.Snapshot, error)

func (f snapshotSourceFunc) Snapshot(ctx context.Context) (deploysystem.Snapshot, error) {
	return f(ctx)
}

func TestPodFilteringSourceSnapshot(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	raw := deploysystem.Snapshot{Host: "node", ObservedAt: time.Now(), Pods: []deploysystem.Pod{{Name: "deny"}, {Name: "keep"}}}
	var sourceErr error
	source := snapshotSourceFunc(func(got context.Context) (deploysystem.Snapshot, error) {
		require.Equal(t, ctx, got)
		return raw, sourceErr
	})
	filter := newFilteringSource(source, &nop.Logger{}, config.PodProfileOptions{
		Default:  true,
		DenyList: []config.DenyListEntry{{Pattern: "["}, {Pattern: "^deny$"}},
	})
	got, err := filter.Snapshot(ctx)
	require.NoError(t, err)
	require.Equal(t, deploysystem.Snapshot{Host: raw.Host, ObservedAt: raw.ObservedAt, Pods: []deploysystem.Pod{{Name: "keep"}}}, got)
	require.Equal(t, []deploysystem.Pod{{Name: "deny"}, {Name: "keep"}}, raw.Pods)

	sourceErr = errors.New("source unavailable")
	_, err = filter.Snapshot(ctx)
	require.ErrorIs(t, err, sourceErr)
}

func TestPodFilterPriority(t *testing.T) {
	var options config.PodProfileOptions
	require.NoError(t, yaml.Unmarshal([]byte(`
blacklist:
  - pattern: '^blocked'
whitelist:
  - pattern: '^redis-0$'
  - pattern: '-allowed$'
`), &options))
	first := deploysystem.Pod{Name: "redis-0", CgroupPath: "pods/a"}
	second := deploysystem.Pod{Name: "redis-0", CgroupPath: "pods/b"}
	raw := deploysystem.Snapshot{Pods: []deploysystem.Pod{{Name: "blocked-allowed"}, first, second, {Name: "other"}}}
	source := snapshotSourceFunc(func(context.Context) (deploysystem.Snapshot, error) { return raw, nil })
	filter := newFilteringSource(source, &nop.Logger{}, options)
	got, err := filter.Snapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, []deploysystem.Pod{first, second}, got.Pods)
}

func TestPodOptionsYAMLKeys(t *testing.T) {
	const legacy = "whitelist: [{pattern: old}]\nblacklist: [{pattern: old}]\n"
	const current = "allowlist: [{pattern: new}]\ndenylist: [{pattern: new, reason: maintenance}]\n"
	const empty = "allowlist: []\ndenylist: []\n"
	var want config.PodProfileOptions
	require.NoError(t, yaml.Unmarshal([]byte(current), &want))
	require.Equal(t, []config.AllowListEntry{{Pattern: "new"}}, want.AllowList)
	require.Equal(t, []config.DenyListEntry{{Pattern: "new", Reason: "maintenance"}}, want.DenyList)

	for _, input := range []string{legacy + current, current + legacy} {
		var got config.PodProfileOptions
		require.NoError(t, yaml.Unmarshal([]byte(input), &got))
		require.Equal(t, want, got)
	}
	for _, input := range []string{legacy + empty, empty + legacy} {
		got := config.PodProfileOptions{Default: true}
		require.NoError(t, yaml.Unmarshal([]byte(input), &got))
		require.Equal(t, []config.AllowListEntry{}, got.AllowList)
		require.Equal(t, []config.DenyListEntry{}, got.DenyList)
		require.True(t, got.Default)
	}

	var mixed config.PodProfileOptions
	require.NoError(t, yaml.Unmarshal([]byte("whitelist: [{pattern: old}]\ndenylist: [{pattern: new}]\ndefault: true"), &mixed))
	require.Equal(t, []config.AllowListEntry{{Pattern: "old"}}, mixed.AllowList)
	require.Equal(t, []config.DenyListEntry{{Pattern: "new"}}, mixed.DenyList)
	require.True(t, mixed.Default)
	data, err := yaml.Marshal(mixed)
	require.NoError(t, err)
	require.YAMLEq(t, "allowlist: [{pattern: old}]\ndenylist: [{pattern: new, reason: ''}]\ndefault: true", string(data))
}
