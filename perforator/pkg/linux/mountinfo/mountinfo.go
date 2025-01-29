package mountinfo

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/yandex/perforator/library/go/core/log"
	"github.com/yandex/perforator/library/go/core/metrics"
	"github.com/yandex/perforator/perforator/pkg/linux/procfs"
)

////////////////////////////////////////////////////////////////////////////////

const (
	MountScanPeriod = time.Minute
)

////////////////////////////////////////////////////////////////////////////////

type mountPointKey struct {
	Device  procfs.Device
	Path    string
	MountID int
}

type MountPoint struct {
	l   log.Logger
	key mountPointKey
}

// Get the file for this mount point.
// There can be ABA problems when mount device & path are reused.
// But the procfs API is racy as hell, so we have to suffer.
func (m *MountPoint) Open() (f *os.File, err error) {
	f, err = os.Open(m.key.Path)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to open mountpoint %s: %w",
			m.key.Path, err,
		)
	}

	defer func() {
		if err != nil {
			err := f.Close()
			if err != nil {
				m.l.Error("Failed to close mount point directory", log.Error(err))
			}
		}
	}()

	stat, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat mountpoint file: %w", err)
	}

	device := uint64(stat.Sys().(*syscall.Stat_t).Dev)
	if device != m.key.Device.Mkdev() {
		return nil, fmt.Errorf("mismatched device number: got %d, expected %d:%d",
			device, m.key.Device.Maj, m.key.Device.Min,
		)
	}

	return f, nil
}

func (m *MountPoint) String() string {
	return fmt.Sprintf("mountpoint %d:%d:%s", m.key.Device.Mkdev(), m.key.MountID, m.key.Path)
}

////////////////////////////////////////////////////////////////////////////////

type Watcher struct {
	mounts     map[int]*MountPoint
	keyToMount map[mountPointKey]*MountPoint
	metrics    mountInfoMetrics

	mutex sync.RWMutex
	l     log.Logger
}

type mountInfoMetrics struct {
	mountPointCount metrics.FuncIntGauge
}

func NewWatcher(l log.Logger, m metrics.Registry) *Watcher {
	watcher := &Watcher{
		mounts:     make(map[int]*MountPoint),
		keyToMount: make(map[mountPointKey]*MountPoint),
		l:          l.WithName("mount_info_scanner"),
	}

	watcher.metrics = mountInfoMetrics{
		mountPointCount: m.WithTags(map[string]string{"kind": "current"}).FuncIntGauge(
			"mount_point.count",
			func() int64 {
				return int64(watcher.MountPointCount())
			},
		),
	}

	return watcher
}

func (m *Watcher) MountPointCount() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.mounts)
}

func (m *Watcher) scan() error {
	mountInfos, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return err
	}
	defer mountInfos.Close()

	s := bufio.NewScanner(bufio.NewReader(mountInfos))
	newMounts := map[int]*MountPoint{}
	newKeyToMount := map[mountPointKey]*MountPoint{}

	for s.Scan() {
		var mountRoot, mountPoint string
		var mountID, parentMountID, maj, min int

		n, err := fmt.Sscanf(s.Text(), "%d %d %d:%d %s %s",
			&mountID, &parentMountID,
			&maj, &min,
			&mountRoot, &mountPoint,
		)

		if n < 6 {
			return fmt.Errorf("failed to parse /proc/self/mountinfo line %q: %w", s.Text(), err)
		}

		key := mountPointKey{
			Device: procfs.Device{
				Min: uint32(min),
				Maj: uint32(maj),
			},
			MountID: mountID,
			Path:    mountPoint,
		}

		mountInfo := m.keyToMount[key]

		if mountInfo != nil {
			m.l.Debug(
				"Keep opened mount point",
				log.String("mount_point", mountPoint),
				log.Int("mount_id", mountID),
				log.String("path", mountPoint),
				log.Int("maj", maj),
				log.Int("min", min),
			)
			newMounts[mountInfo.key.MountID] = mountInfo
			newKeyToMount[mountInfo.key] = mountInfo
			continue
		}

		m.l.Debug(
			"Found new mount point",
			log.String("mount_point", mountPoint),
			log.Int("maj", maj),
			log.Int("min", min),
			log.Int("mount_id", mountID),
		)

		newMountPoint := &MountPoint{m.l, key}
		newMounts[mountID] = newMountPoint
		newKeyToMount[key] = newMountPoint
	}

	m.mutex.Lock()
	m.mounts = newMounts
	m.keyToMount = newKeyToMount
	m.mutex.Unlock()

	return nil
}

func (m *Watcher) GetMountPoint(mountID int) *MountPoint {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.mounts[mountID]
}

func (m *Watcher) RunPoller(ctx context.Context) error {
	tick := time.NewTicker(MountScanPeriod)
	for {
		m.l.Info("Run mount info scanner")
		err := m.scan()
		if err != nil {
			m.l.Error("Failed to update mount info", log.Error(err))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-tick.C:
		}
	}
}

////////////////////////////////////////////////////////////////////////////////
