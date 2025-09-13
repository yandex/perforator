// Copyright 2019 The Prometheus Authors
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

// Package btrfs provides access to statistics exposed by ext4 filesystems.
package ext4

import (
	"path/filepath"
	"strings"

	"github.com/prometheus/procfs/internal/fs"
	"github.com/prometheus/procfs/internal/util"
)

const (
	sysFSPath     = "fs"
	sysFSExt4Path = "ext4"
)

// Stats contains statistics for a single Btrfs filesystem.
// See Linux fs/btrfs/sysfs.c for more information.
type Stats struct {
	Name string

	Errors   uint64
	Warnings uint64
	Messages uint64
}

// FS represents the pseudo-filesystems proc and sys, which provides an
// interface to kernel data structures.
type FS struct {
	proc *fs.FS
	sys  *fs.FS
}

// NewDefaultFS returns a new blockdevice fs using the default mountPoints for proc and sys.
// It will error if either of these mount points can't be read.
func NewDefaultFS() (FS, error) {
	return NewFS(fs.DefaultProcMountPoint, fs.DefaultSysMountPoint)
}

// NewFS returns a new XFS handle using the given proc and sys mountPoints. It will error
// if either of the mounts point can't be read.
func NewFS(procMountPoint string, sysMountPoint string) (FS, error) {
	if strings.TrimSpace(procMountPoint) == "" {
		procMountPoint = fs.DefaultProcMountPoint
	}
	procfs, err := fs.NewFS(procMountPoint)
	if err != nil {
		return FS{}, err
	}
	if strings.TrimSpace(sysMountPoint) == "" {
		sysMountPoint = fs.DefaultSysMountPoint
	}
	sysfs, err := fs.NewFS(sysMountPoint)
	if err != nil {
		return FS{}, err
	}
	return FS{&procfs, &sysfs}, nil
}

// ProcStat returns stats for the filesystem.
func (fs FS) ProcStat() ([]*Stats, error) {
	matches, err := filepath.Glob(fs.sys.Path("fs/ext4/*"))
	if err != nil {
		return nil, err
	}

	stats := make([]*Stats, 0, len(matches))
	for _, m := range matches {
		s := &Stats{}

		// "*" used in glob above indicates the name of the filesystem.
		name := filepath.Base(m)
		s.Name = name
		for file, p := range map[string]*uint64{
			"errors_count":  &s.Errors,
			"warning_count": &s.Warnings,
			"msg_count":     &s.Messages,
		} {
			var val uint64
			val, err = util.ReadUintFromFile(fs.sys.Path(sysFSPath, sysFSExt4Path, name, file))
			if err == nil {
				*p = val
			}
		}

		stats = append(stats, s)
	}

	return stats, nil
}
