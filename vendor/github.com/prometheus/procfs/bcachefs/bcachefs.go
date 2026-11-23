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

// Package bcachefs provides access to statistics exposed by Bcachefs filesystems.
package bcachefs

// Stats contains statistics for a single Bcachefs filesystem.
type Stats struct {
	UUID                string
	BtreeCacheSizeBytes uint64

	Compression map[string]CompressionStats
	Errors      map[string]ErrorStats
	Counters    map[string]CounterStats
	BtreeWrites map[string]BtreeWriteStats
	Devices     map[string]*DeviceStats
}

// CompressionStats contains compression statistics for a specific algorithm.
type CompressionStats struct {
	CompressedBytes        uint64
	UncompressedBytes      uint64
	AverageExtentSizeBytes uint64
}

// ErrorStats contains error count and timestamp for a specific error type.
type ErrorStats struct {
	Count     uint64
	Timestamp uint64
}

// CounterStats contains counter values since mount and since filesystem creation.
type CounterStats struct {
	SinceMount              uint64
	SinceFilesystemCreation uint64
}

// BtreeWriteStats contains btree write statistics for a specific type.
type BtreeWriteStats struct {
	Count     uint64
	SizeBytes uint64
}

// DeviceStats contains statistics for a Bcachefs device.
type DeviceStats struct {
	Label           string
	State           string
	BucketSizeBytes uint64
	Buckets         uint64
	Durability      uint64
	IODone          map[string]map[string]uint64
	IOErrors        map[string]uint64
}
