package dso

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics/mock"
	"github.com/yandex/perforator/perforator/pkg/linux"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
	"github.com/yandex/perforator/perforator/pkg/xelf"
	"github.com/yandex/perforator/perforator/pkg/xlog"
)

func TestRegistry_Simple(t *testing.T) {
	logger := xlog.ForTest(t)
	ctx := context.Background()

	registry, err := NewRegistry(logger, mock.NewRegistry(nil), nil, nil)
	require.NoError(t, err)

	buildID := "abacaba"
	buildInfo := &xelf.BuildInfo{BuildID: buildID}

	registry.register(ctx, buildInfo, nil)
	require.Equal(t, buildID, registry.get(buildID).buildInfo.BuildID)

	registry.register(ctx, buildInfo, nil)
	require.Equal(t, 1, registry.getMappingCount())

	registry.release(ctx, buildID)
	require.Equal(t, 1, registry.getMappingCount())
	require.Equal(t, buildID, registry.get(buildID).buildInfo.BuildID)

	registry.release(ctx, buildID)
	require.Equal(t, 0, registry.getMappingCount())

	// no-op
	registry.release(ctx, buildID)

	var expected *DSO = nil
	require.Equal(t, expected, registry.get(buildID))
}

func TestRegistry_Concurrent(t *testing.T) {
	l := xlog.ForTest(t)

	registry, err := NewRegistry(l, mock.NewRegistry(nil), nil, nil)
	require.NoError(t, err)

	buildIDs := []string{"a", "b", "c", "d", "e"}
	iterations := 10000

	g, ctx := errgroup.WithContext(context.Background())

	addMappings := func() error {
		for i := 0; i < iterations; i++ {
			buildID := buildIDs[i%len(buildIDs)]
			buildInfo := &xelf.BuildInfo{BuildID: buildID}
			registry.register(ctx, buildInfo, nil)
		}

		return nil
	}

	deleteMappings := func() error {
		for i := iterations - 1; i >= 0; i-- {
			buildID := buildIDs[i%len(buildIDs)]
			registry.release(ctx, buildID)
		}

		return nil
	}

	g.Go(addMappings)
	err = g.Wait()
	require.NoError(t, err)

	for _, buildID := range buildIDs {
		require.Equal(t, buildID, registry.get(buildID).buildInfo.BuildID)
	}
	g.Go(deleteMappings)
	err = g.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, registry.getMappingCount())

	// concurrent addMappings
	g.Go(addMappings)
	g.Go(addMappings)
	err = g.Wait()
	require.NoError(t, err)
	require.Equal(t, len(buildIDs), registry.getMappingCount())

	// concurrent deleteMappings
	g.Go(deleteMappings)
	g.Go(deleteMappings)
	err = g.Wait()
	require.NoError(t, err)

	require.Equal(t, 0, registry.getMappingCount())

	// concurrent everything
	g.Go(addMappings)
	g.Go(deleteMappings)
	g.Go(addMappings)
	g.Go(deleteMappings)

	err = g.Wait()
	require.NoError(t, err)
}

func TestStorage_Simple(t *testing.T) {
	logger := xlog.ForTest(t)
	ctx := context.Background()
	storage, err := NewStorage(
		logger,
		mock.NewRegistry(
			&mock.RegistryOpts{
				AllowLoadRegisteredMetrics: true,
			},
		),
		nil,
		nil,
	)
	require.NoError(t, err)

	storage.AddProcess(0)
	storage.AddProcess(1)
	storage.AddProcess(2)
	storage.AddProcess(3)

	_, err = storage.ResolveAddress(ctx, 15, 0)
	require.Error(t, err)

	_, err = storage.ResolveAddress(ctx, 0, 0xdeadadd7e55)
	require.Error(t, err)
}

func TestStorage_OneMapping(t *testing.T) {
	logger := xlog.ForTest(t)
	ctx := context.Background()
	storage, err := NewStorage(
		logger,
		mock.NewRegistry(
			&mock.RegistryOpts{
				AllowLoadRegisteredMetrics: true,
			},
		),
		nil,
		nil,
	)
	require.NoError(t, err)

	storage.AddMapping(
		ctx,
		123,
		Mapping{Mapping: procfs.Mapping{
			Begin:  0,
			End:    1024,
			Path:   "legolas.elf",
			Inode:  procfs.Inode{ID: 0x140de, Gen: 0},
			Offset: 2048,
		}},
		nil,
	)
	var loc *Location

	_, err = storage.ResolveAddress(ctx, 123, 0xdeadadd7e55)
	require.ErrorContains(t, err, "points to unknown mapping")

	loc, err = storage.ResolveAddress(ctx, 123, 0)
	require.NoError(t, err)
	require.Equal(t, loc, &Location{
		Path:   "legolas.elf",
		Inode:  procfs.Inode{ID: 0x140de, Gen: 0},
		Offset: 2048,
	})

	loc, err = storage.ResolveAddress(ctx, 123, 512)
	require.NoError(t, err)
	require.Equal(t, loc, &Location{
		Path:   "legolas.elf",
		Inode:  procfs.Inode{ID: 0x140de, Gen: 0},
		Offset: 2048 + 512,
	})

	loc, err = storage.ResolveAddress(ctx, 123, 1023)
	require.NoError(t, err)
	require.Equal(t, loc, &Location{
		Path:   "legolas.elf",
		Inode:  procfs.Inode{ID: 0x140de, Gen: 0},
		Offset: 3071,
	})

	_, err = storage.ResolveAddress(ctx, 123, 1024)
	require.ErrorContains(t, err, "points to unknown mapping")
}

func TestStorage_OverlappingMappings(t *testing.T) {
	logger := xlog.ForTest(t)
	ctx := context.Background()
	storage, err := NewStorage(
		logger,
		mock.NewRegistry(
			&mock.RegistryOpts{
				AllowLoadRegisteredMetrics: true,
			},
		),
		nil,
		nil,
	)
	require.NoError(t, err)

	storage.AddMapping(
		ctx,
		linux.CurrentNamespacePID(0),
		Mapping{
			Mapping: procfs.Mapping{
				Begin: 0,
				End:   1024,
				Inode: procfs.Inode{ID: 0},
			},
			BuildInfo: &xelf.BuildInfo{
				BuildID: "a",
			},
		},
		nil,
	)
	storage.AddMapping(
		ctx,
		linux.CurrentNamespacePID(0),
		Mapping{
			Mapping: procfs.Mapping{
				Begin: 8000,
				End:   10000,
				Inode: procfs.Inode{ID: 2},
			},
			BuildInfo: &xelf.BuildInfo{
				BuildID: "c",
			},
		},
		nil,
	)

	storage.AddMapping(
		ctx,
		linux.CurrentNamespacePID(0),
		Mapping{
			Mapping: procfs.Mapping{
				Begin: 512,
				End:   4096,
				Inode: procfs.Inode{ID: 1},
			},
			BuildInfo: &xelf.BuildInfo{
				BuildID: "b",
			},
		},
		nil,
	)

	loc, err := storage.ResolveAddress(ctx, linux.CurrentNamespacePID(0), 600)
	require.NoError(t, err)
	require.Equal(t, &Location{Inode: procfs.Inode{ID: 1}, Offset: 600 - 512}, loc)

	_, err = storage.ResolveAddress(ctx, linux.CurrentNamespacePID(0), 200)
	require.Error(t, err)

	loc, err = storage.ResolveAddress(ctx, linux.CurrentNamespacePID(0), 9000)
	require.NoError(t, err)
	require.Equal(t, &Location{Inode: procfs.Inode{ID: 2}, Offset: 9000 - 8000}, loc)
}

func TestStorage_MultipleProcsAndMappings(t *testing.T) {
	logger := xlog.ForTest(t)
	ctx := context.Background()
	storage, err := NewStorage(
		logger,
		mock.NewRegistry(
			&mock.RegistryOpts{
				AllowLoadRegisteredMetrics: true,
			},
		),
		nil,
		nil,
	)
	require.NoError(t, err)

	mappings := []Mapping{
		{
			Mapping: procfs.Mapping{
				Begin:  0,
				End:    1024,
				Offset: 1,
				Inode:  procfs.Inode{ID: 1},
			},
			BuildInfo: &xelf.BuildInfo{
				BuildID: "a",
			},
		},
		{
			Mapping: procfs.Mapping{
				Begin:  2048,
				End:    4096,
				Offset: 148,
				Inode:  procfs.Inode{ID: 2},
			},
			BuildInfo: &xelf.BuildInfo{
				BuildID: "b",
			},
		},
		{
			Mapping: procfs.Mapping{
				Begin:  8000,
				End:    10000,
				Offset: 73,
				Inode:  procfs.Inode{ID: 3},
			},
			BuildInfo: &xelf.BuildInfo{
				BuildID: "c",
			},
		},
	}
	for _, mapping := range mappings {
		storage.AddMapping(ctx, linux.CurrentNamespacePID(0), mapping, nil)
	}

	loc, err := storage.ResolveAddress(ctx, linux.CurrentNamespacePID(0), 8400)
	require.NoError(t, err)
	require.Equal(t, &Location{Inode: procfs.Inode{ID: 3}, Offset: 8400 - 8000 + 73}, loc)

	loc, err = storage.ResolveAddress(ctx, linux.CurrentNamespacePID(0), 512)
	require.NoError(t, err)
	require.Equal(t, &Location{Inode: procfs.Inode{ID: 1}, Offset: 512 - 0 + 1}, loc)

	loc, err = storage.ResolveAddress(ctx, linux.CurrentNamespacePID(0), 3000)
	require.NoError(t, err)
	require.Equal(t, &Location{Inode: procfs.Inode{ID: 2}, Offset: 3000 - 2048 + 148}, loc)

	require.Equal(t, len(mappings), storage.registry.getMappingCount())

	newMappings := []Mapping{mappings[0], mappings[1]}
	for _, mapping := range newMappings {
		storage.AddMapping(ctx, linux.CurrentNamespacePID(1), mapping, nil)
	}

	loc, err = storage.ResolveAddress(ctx, linux.CurrentNamespacePID(1), 3012)
	require.NoError(t, err)
	require.Equal(t, &Location{Inode: procfs.Inode{ID: 2}, Offset: 3012 - 2048 + 148}, loc)

	storage.RemoveProcess(ctx, linux.CurrentNamespacePID(0))
	require.Equal(t, 1, storage.GetProcessCount())
	require.Equal(t, len(newMappings), storage.registry.getMappingCount())

	for _, mapping := range newMappings {
		require.Equal(t, mapping.BuildInfo.BuildID, storage.registry.get(mapping.BuildInfo.BuildID).buildInfo.BuildID)
	}

	loc, err = storage.ResolveAddress(ctx, linux.CurrentNamespacePID(1), 670)
	require.NoError(t, err)
	require.Equal(t, &Location{Inode: procfs.Inode{ID: 1}, Offset: 670 - 0 + 1}, loc)

	storage.RemoveProcess(ctx, linux.CurrentNamespacePID(1))
	require.Equal(t, 0, storage.registry.getMappingCount())
	require.Equal(t, 0, storage.GetProcessCount())
}

func TestStorage_SameMappings(t *testing.T) {
	logger := xlog.ForTest(t)
	ctx := context.Background()
	storage, err := NewStorage(
		logger,
		mock.NewRegistry(
			&mock.RegistryOpts{
				AllowLoadRegisteredMetrics: true,
			},
		),
		nil,
		nil,
	)
	require.NoError(t, err)

	mapping := Mapping{
		Mapping: procfs.Mapping{
			Begin:  0,
			End:    1024,
			Offset: 1,
			Inode:  procfs.Inode{ID: 1},
			Path:   "/first/mapping/file",
		},
		BuildInfo: &xelf.BuildInfo{
			BuildID: "aaa",
		},
	}
	mapping2 := Mapping{
		Mapping: procfs.Mapping{
			Begin:  1025,
			End:    2049,
			Offset: 1,
			Inode:  procfs.Inode{ID: 2},
			Path:   "/second/mapping/file",
		},
		BuildInfo: &xelf.BuildInfo{
			BuildID: "bbb",
		},
	}

	for range 5 {
		storage.AddMapping(ctx, linux.CurrentNamespacePID(0), mapping, nil)
		storage.AddMapping(ctx, linux.CurrentNamespacePID(0), mapping2, nil)
	}

	logger.Debug(ctx, "Compactify process", log.Int("pid", 0))
	require.Equal(t, 2, storage.Compactify(ctx, 0))

	logger.Debug(ctx, "Removing process", log.Int("pid", 0))
	storage.RemoveProcess(ctx, 0)

	require.Equal(t, 0, storage.registry.getMappingCount())
}

func TestStorage_Concurrent(t *testing.T) {
	logger := xlog.ForTest(t)
	storage, err := NewStorage(
		logger,
		mock.NewRegistry(
			&mock.RegistryOpts{
				AllowLoadRegisteredMetrics: true,
			},
		),
		nil,
		nil,
	)
	require.NoError(t, err)

	mappings := []Mapping{}
	for i := uint64(0); i < 100; i++ {
		mappings = append(
			mappings,
			Mapping{
				Mapping: procfs.Mapping{
					Begin: i * 1024,
					End:   (i + 1) * 1024,
					Inode: procfs.Inode{ID: i},
				},
				BuildInfo: &xelf.BuildInfo{
					BuildID: fmt.Sprintf("%d", i),
				},
			},
		)
	}

	g, ctx := errgroup.WithContext(context.Background())
	for i := uint64(0); i < 1000; i++ {
		pid := i
		g.Go(func() error {
			for j := pid % 100; j < pid%100+10; j++ {
				if j >= uint64(len(mappings)) {
					continue
				}

				storage.AddMapping(ctx, linux.CurrentNamespacePID(pid), mappings[j], nil)

				if storage.registry.get(mappings[j].BuildInfo.BuildID) == nil {
					return fmt.Errorf("no mapping `%s` found in dso map", mappings[j].BuildInfo.BuildID)
				}
			}

			for j := pid % 100; j < pid%100+10; j++ {
				if j >= uint64(len(mappings)) {
					continue
				}

				if storage.registry.get(mappings[j].BuildInfo.BuildID) == nil {
					return fmt.Errorf("no mapping `%s` found in dso map", mappings[j].BuildInfo.BuildID)
				}
			}

			storage.RemoveProcess(ctx, linux.CurrentNamespacePID(pid))

			return nil
		})
	}

	err = g.Wait()
	require.NoError(t, err)
	require.Equal(t, 0, storage.registry.getMappingCount())
	require.Equal(t, 0, storage.GetProcessCount())

	for i := uint64(0); i < 1000; i++ {
		pid := i
		g.Go(func() error {
			for j := pid % 100; j < pid%100+10; j++ {
				if j >= uint64(len(mappings)) {
					continue
				}

				storage.AddMapping(ctx, linux.CurrentNamespacePID(pid), mappings[j], nil)
			}

			return nil
		})
		g.Go(func() error {
			storage.RemoveProcess(ctx, linux.CurrentNamespacePID(pid))
			return nil
		})
	}
}

func TestPreparedMappingsDoNotChangeProcessUntilCommit(t *testing.T) {
	ctx := t.Context()
	storage, err := NewStorage(xlog.ForTest(t), mock.NewRegistry(nil), nil, nil)
	require.NoError(t, err)
	old := Mapping{Mapping: procfs.Mapping{Begin: 1, End: 10, Path: "old"}, BuildInfo: &xelf.BuildInfo{BuildID: "old"}}
	storage.AddMapping(ctx, 42, old, nil)
	prepared := []*Mapping{{Mapping: procfs.Mapping{Begin: 1, End: 10, Path: "new"}, BuildInfo: &xelf.BuildInfo{BuildID: "new"}}}
	storage.PrepareMapping(ctx, prepared[0], nil)
	mapping, err := storage.ResolveMapping(ctx, 42, 2)
	require.NoError(t, err)
	require.Equal(t, "old", mapping.Path)
	storage.ReplaceMappings(ctx, 42, prepared)
	require.Nil(t, storage.registry.get("old"))
	require.NotNil(t, storage.registry.get("new"))
	mapping, err = storage.ResolveMapping(ctx, 42, 2)
	require.NoError(t, err)
	require.Equal(t, "new", mapping.Path)
	storage.RemoveProcess(ctx, 42)
	require.Zero(t, storage.registry.getMappingCount())
}

func TestDiscardingPreparedMappingsPreservesCommittedReferences(t *testing.T) {
	ctx := t.Context()
	storage, err := NewStorage(xlog.ForTest(t), mock.NewRegistry(nil), nil, nil)
	require.NoError(t, err)
	mapping := Mapping{Mapping: procfs.Mapping{Begin: 1, End: 10}, BuildInfo: &xelf.BuildInfo{BuildID: "shared"}}
	storage.AddMapping(ctx, 42, mapping, nil)
	storage.PrepareMapping(ctx, &mapping, nil)
	storage.ReleaseMappings(ctx, []*Mapping{&mapping})
	require.NotNil(t, storage.registry.get("shared"))
	storage.RemoveProcess(ctx, 42)
	require.Zero(t, storage.registry.getMappingCount())
}
