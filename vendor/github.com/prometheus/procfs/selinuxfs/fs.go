// Copyright 2023 The Prometheus Authors
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

//go:build linux && !noselinux
// +build linux,!noselinux

package selinuxfs

import (
	"github.com/prometheus/procfs/internal/fs"
)

// FS represents the pseudo-filesystem selinixfs, which provides an interface to
// SELinux data structures.
type FS struct {
	selinux fs.FS
}

// DefaultMountPoint is the common mount point of the selinuxfs filesystem.
const DefaultMountPoint = fs.DefaultSelinuxMountPoint

// NewDefaultFS returns a new FS mounted under the default mountPoint. It will error
// if the mount point can't be read.
func NewDefaultFS() (FS, error) {
	return NewFS(DefaultMountPoint)
}

// NewFS returns a new FS mounted under the given mountPoint. It will error
// if the mount point can't be read.
func NewFS(mountPoint string) (FS, error) {
	fs, err := fs.NewFS(mountPoint)
	if err != nil {
		return FS{}, err
	}
	return FS{fs}, nil
}
