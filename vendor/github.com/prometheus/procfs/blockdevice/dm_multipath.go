// Copyright The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package blockdevice

import (
	"fmt"
	"os"
	"strings"

	"github.com/prometheus/procfs"
	"github.com/prometheus/procfs/internal/util"
)

// DMMultipathDevice contains information about a single DM-multipath device
// discovered by scanning /sys/block/dm-* entries whose dm/uuid starts with
// "mpath-".
type DMMultipathDevice struct {
	// Name is the device-mapper name (from dm/name), e.g. "mpathA".
	Name string
	// SysfsName is the kernel block device name, e.g. "dm-5".
	SysfsName string
	// UUID is the full DM UUID string, e.g. "mpath-360000000000001".
	UUID string
	// Suspended is true when dm/suspended reads "1".
	Suspended bool
	// SizeBytes is the device size in bytes (sectors × 512).
	SizeBytes uint64
	// Paths lists the underlying block devices from the slaves/ directory.
	Paths []DMMultipathPath
}

// DMMultipathPath represents one underlying path device for a DM-multipath map.
type DMMultipathPath struct {
	// Device is the block device name, e.g. "sdi".
	Device string
	// State is the raw device state read from
	// /sys/block/<device>/device/state, e.g. "running", "offline", "live".
	State string
}

// DMMultipathDevices discovers DM-multipath devices by scanning
// /sys/block/dm-* and filtering on dm/uuid prefix "mpath-".
//
// It returns a slice of DMMultipathDevice structs. If no multipath devices
// are found, it returns an empty (non-nil) slice and no error.
func (fs FS) DMMultipathDevices() ([]DMMultipathDevice, error) {
	blockDir := fs.sys.Path(sysBlockPath)

	entries, err := os.ReadDir(blockDir)
	if err != nil {
		return nil, err
	}

	devices := make([]DMMultipathDevice, 0)
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "dm-") {
			continue
		}

		uuid, err := util.SysReadFile(fs.sys.Path(sysBlockPath, entry.Name(), sysBlockDM, "uuid"))
		if err != nil {
			// dm/uuid missing means this is not a device-mapper device; skip it.
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read dm/uuid for %s: %w", entry.Name(), err)
		}
		if !strings.HasPrefix(uuid, "mpath-") {
			continue
		}

		name, err := util.SysReadFile(fs.sys.Path(sysBlockPath, entry.Name(), sysBlockDM, "name"))
		if err != nil {
			return nil, fmt.Errorf("failed to read dm/name for %s: %w", entry.Name(), err)
		}

		suspendedVal, err := util.ReadUintFromFile(fs.sys.Path(sysBlockPath, entry.Name(), sysBlockDM, "suspended"))
		if err != nil {
			return nil, fmt.Errorf("failed to read dm/suspended for %s: %w", entry.Name(), err)
		}

		sectors, err := util.ReadUintFromFile(fs.sys.Path(sysBlockPath, entry.Name(), sysBlockSize))
		if err != nil {
			return nil, fmt.Errorf("failed to read size for %s: %w", entry.Name(), err)
		}

		paths, err := fs.dmMultipathPaths(entry.Name())
		if err != nil {
			return nil, err
		}

		devices = append(devices, DMMultipathDevice{
			Name:      name,
			SysfsName: entry.Name(),
			UUID:      uuid,
			Suspended: suspendedVal == 1,
			SizeBytes: sectors * procfs.SectorSize,
			Paths:     paths,
		})
	}

	return devices, nil
}

// dmMultipathPaths reads the slaves/ directory of a dm device and returns
// the path devices with their states.
func (fs FS) dmMultipathPaths(dmDevice string) ([]DMMultipathPath, error) {
	slavesDir := fs.sys.Path(sysBlockPath, dmDevice, sysUnderlyingDev)

	entries, err := os.ReadDir(slavesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	paths := make([]DMMultipathPath, 0, len(entries))
	for _, entry := range entries {
		state, err := util.SysReadFile(fs.sys.Path(sysBlockPath, entry.Name(), sysDevicePath, "state"))
		if err != nil {
			return nil, fmt.Errorf("failed to read device/state for %s: %w", entry.Name(), err)
		}
		paths = append(paths, DMMultipathPath{
			Device: entry.Name(),
			State:  state,
		})
	}

	return paths, nil
}
