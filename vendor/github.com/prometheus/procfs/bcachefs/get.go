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

package bcachefs

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/prometheus/procfs/internal/fs"
	"github.com/prometheus/procfs/internal/util"
)

// FS represents the pseudo-filesystem sys, which provides an interface to
// kernel data structures.
type FS struct {
	sys *fs.FS
}

// NewDefaultFS returns a new Bcachefs using the default sys fs mount point. It will error
// if the mount point can't be read.
func NewDefaultFS() (FS, error) {
	return NewFS(fs.DefaultSysMountPoint)
}

// NewFS returns a new Bcachefs filesystem using the given sys fs mount point. It will error
// if the mount point can't be read.
func NewFS(mountPoint string) (FS, error) {
	if strings.TrimSpace(mountPoint) == "" {
		mountPoint = fs.DefaultSysMountPoint
	}
	sys, err := fs.NewFS(mountPoint)
	if err != nil {
		return FS{}, err
	}
	return FS{&sys}, nil
}

// Stats retrieves Bcachefs filesystem runtime statistics for each mounted Bcachefs filesystem.
func (fs FS) Stats() ([]*Stats, error) {
	base := fs.sys.Path("fs/bcachefs")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}

	stats := make([]*Stats, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		uuidPath := filepath.Join(base, entry.Name())
		s, err := GetStats(uuidPath)
		if err != nil {
			return nil, err
		}
		s.UUID = entry.Name()
		stats = append(stats, s)
	}

	return stats, nil
}

// GetStats collects all Bcachefs statistics from sysfs.
func GetStats(uuidPath string) (*Stats, error) {
	r := &reader{path: uuidPath}
	s := r.readFilesystemStats()
	if r.err != nil {
		return nil, r.err
	}
	return s, nil
}

type reader struct {
	path string
	err  error
}

// readFile reads a file relative to the path of the reader.
// Non-existing files are ignored.
func (r *reader) readFile(n string) string {
	if r.err != nil {
		return ""
	}
	b, err := util.ReadFileNoStat(filepath.Join(r.path, n))
	if err != nil {
		if !os.IsNotExist(err) {
			r.err = err
		}
		return ""
	}
	return strings.TrimSpace(string(b))
}

func (r *reader) readHumanBytes(n string) uint64 {
	s := r.readFile(n)
	if r.err != nil || s == "" {
		return 0
	}
	v, err := parseHumanReadableBytes(s)
	if err != nil {
		r.err = err
		return 0
	}
	return v
}

func (r *reader) readFilesystemStats() *Stats {
	stats := &Stats{
		Compression: make(map[string]CompressionStats),
		Errors:      make(map[string]ErrorStats),
		Counters:    make(map[string]CounterStats),
		BtreeWrites: make(map[string]BtreeWriteStats),
		Devices:     make(map[string]*DeviceStats),
	}

	stats.BtreeCacheSizeBytes = r.readHumanBytes("btree_cache_size")

	if r.err != nil {
		return stats
	}

	comp, err := parseCompressionStats(filepath.Join(r.path, "compression_stats"))
	if err != nil {
		r.err = err
		return stats
	}
	stats.Compression = comp

	errs, err := parseErrors(filepath.Join(r.path, "errors"))
	if err != nil {
		r.err = err
		return stats
	}
	stats.Errors = errs

	counters, err := parseCounters(filepath.Join(r.path, "counters"))
	if err != nil {
		r.err = err
		return stats
	}
	stats.Counters = counters

	writes, err := parseBtreeWriteStats(filepath.Join(r.path, "btree_write_stats"))
	if err != nil {
		r.err = err
		return stats
	}
	stats.BtreeWrites = writes

	devices, err := parseDevices(r.path)
	if err != nil {
		r.err = err
		return stats
	}
	stats.Devices = devices

	return stats
}

func parseHumanReadableBytes(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}

	multiplier := float64(1)
	lastChar := s[len(s)-1]
	switch lastChar {
	case 'k', 'K':
		multiplier = 1024
		s = s[:len(s)-1]
	case 'm', 'M':
		multiplier = 1024 * 1024
		s = s[:len(s)-1]
	case 'g', 'G':
		multiplier = 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 't', 'T':
		multiplier = 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 'p', 'P':
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	case 'e', 'E':
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024 * 1024
		s = s[:len(s)-1]
	}

	value, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0, err
	}
	return uint64(value * multiplier), nil
}

func parseCompressionStats(path string) (map[string]CompressionStats, error) {
	file, err := openIfExists(path)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return map[string]CompressionStats{}, nil
	}
	defer file.Close()

	stats := make(map[string]CompressionStats)

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if lineNum == 1 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		algorithm := strings.TrimSuffix(fields[0], ":")
		compressed, err := parseHumanReadableBytes(fields[1])
		if err != nil {
			return nil, err
		}
		uncompressed, err := parseHumanReadableBytes(fields[2])
		if err != nil {
			return nil, err
		}
		var avgExtent uint64
		if len(fields) >= 4 {
			avgExtent, err = parseHumanReadableBytes(fields[3])
			if err != nil {
				return nil, err
			}
		}
		stats[algorithm] = CompressionStats{
			CompressedBytes:        compressed,
			UncompressedBytes:      uncompressed,
			AverageExtentSizeBytes: avgExtent,
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func parseErrors(path string) (map[string]ErrorStats, error) {
	file, err := openIfExists(path)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return map[string]ErrorStats{}, nil
	}
	defer file.Close()

	stats := make(map[string]ErrorStats)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		count, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, err
		}
		var ts uint64
		if len(fields) >= 3 {
			ts, err = strconv.ParseUint(fields[2], 10, 64)
			if err != nil {
				return nil, err
			}
		}
		stats[fields[0]] = ErrorStats{Count: count, Timestamp: ts}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func parseCounters(countersPath string) (map[string]CounterStats, error) {
	entries, err := os.ReadDir(countersPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	stats := make(map[string]CounterStats, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		counterPath := filepath.Join(countersPath, entry.Name())
		counter, err := parseCounterFile(counterPath)
		if err != nil {
			return nil, err
		}
		stats[entry.Name()] = counter
	}

	return stats, nil
}

func parseCounterFile(path string) (CounterStats, error) {
	file, err := openIfExists(path)
	if err != nil || file == nil {
		return CounterStats{}, err
	}
	defer file.Close()

	var stats CounterStats
	var seenCreation bool
	var seenMount bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if valueStr, ok := strings.CutPrefix(line, "since mount:"); ok {
			value, err := parseHumanReadableBytes(valueStr)
			if err != nil {
				return CounterStats{}, err
			}
			stats.SinceMount = value
			seenMount = true
			continue
		}
		if valueStr, ok := strings.CutPrefix(line, "since filesystem creation:"); ok {
			value, err := parseHumanReadableBytes(valueStr)
			if err != nil {
				return CounterStats{}, err
			}
			stats.SinceFilesystemCreation = value
			seenCreation = true
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		return CounterStats{}, err
	}
	if !seenCreation && !seenMount {
		return CounterStats{}, fmt.Errorf("counter file format not recognized")
	}
	return stats, nil
}

func parseBtreeWriteStats(path string) (map[string]BtreeWriteStats, error) {
	file, err := openIfExists(path)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return map[string]BtreeWriteStats{}, nil
	}
	defer file.Close()

	stats := make(map[string]BtreeWriteStats)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if lineNum == 1 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		writeType := strings.TrimSuffix(fields[0], ":")
		count, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return nil, err
		}
		size, err := parseHumanReadableBytes(fields[2])
		if err != nil {
			return nil, err
		}
		stats[writeType] = BtreeWriteStats{Count: count, SizeBytes: size}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func parseDevices(fsPath string) (map[string]*DeviceStats, error) {
	entries, err := os.ReadDir(fsPath)
	if err != nil {
		return nil, err
	}

	devices := make(map[string]*DeviceStats)
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "dev-") {
			continue
		}
		device := strings.TrimPrefix(entry.Name(), "dev-")
		devPath := filepath.Join(fsPath, entry.Name())

		stats := &DeviceStats{
			Label:    readSysfsFile(filepath.Join(devPath, "label")),
			State:    readSysfsFile(filepath.Join(devPath, "state")),
			IODone:   make(map[string]map[string]uint64),
			IOErrors: make(map[string]uint64),
		}

		if bucketSizeRaw := readSysfsFile(filepath.Join(devPath, "bucket_size")); bucketSizeRaw != "" {
			bucketSize, err := parseHumanReadableBytes(bucketSizeRaw)
			if err != nil {
				return nil, err
			}
			stats.BucketSizeBytes = bucketSize
		}

		nbuckets, err := readUintFile(filepath.Join(devPath, "nbuckets"))
		if err != nil {
			return nil, err
		}
		stats.Buckets = nbuckets

		durability, err := readUintFile(filepath.Join(devPath, "durability"))
		if err != nil {
			return nil, err
		}
		stats.Durability = durability

		ioDone, err := parseDeviceIODone(filepath.Join(devPath, "io_done"))
		if err != nil {
			return nil, err
		}
		stats.IODone = ioDone

		ioErrors, err := parseDeviceIOErrors(filepath.Join(devPath, "io_errors"))
		if err != nil {
			return nil, err
		}
		stats.IOErrors = ioErrors

		devices[device] = stats
	}

	return devices, nil
}

func readSysfsFile(path string) string {
	data, err := util.ReadFileNoStat(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func readUintFile(path string) (uint64, error) {
	data, err := util.ReadFileNoStat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	return strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
}

func parseDeviceIODone(path string) (map[string]map[string]uint64, error) {
	file, err := openIfExists(path)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return map[string]map[string]uint64{}, nil
	}
	defer file.Close()

	stats := make(map[string]map[string]uint64)
	scanner := bufio.NewScanner(file)
	var currentOp string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "read:" || line == "write:" {
			currentOp = strings.TrimSuffix(line, ":")
			if _, ok := stats[currentOp]; !ok {
				stats[currentOp] = make(map[string]uint64)
			}
			continue
		}
		if currentOp == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		dataType := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			return nil, err
		}
		stats[currentOp][dataType] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func parseDeviceIOErrors(path string) (map[string]uint64, error) {
	file, err := openIfExists(path)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return map[string]uint64{}, nil
	}
	defer file.Close()

	stats := make(map[string]uint64)
	scanner := bufio.NewScanner(file)
	inCreationSection := false

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "IO errors since filesystem creation") {
			inCreationSection = true
			continue
		}
		if strings.HasPrefix(line, "IO errors since ") {
			break
		}
		if !inCreationSection {
			continue
		}
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		parts := strings.SplitN(trimmed, ":", 2)
		if len(parts) != 2 {
			continue
		}
		errorType := strings.TrimSpace(parts[0])
		valueStr := strings.TrimSpace(parts[1])
		value, err := strconv.ParseUint(valueStr, 10, 64)
		if err != nil {
			return nil, err
		}
		stats[errorType] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func openIfExists(path string) (*os.File, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return file, nil
}
