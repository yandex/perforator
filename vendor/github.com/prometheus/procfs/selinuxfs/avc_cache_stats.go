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
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// SELinux access vector cache statistics.
type AVCStat struct {
	// Number of total lookups
	Lookups uint64
	// Number of total hits
	Hits uint64
	// Number of total misses
	Misses uint64
	// Number of total allocations
	Allocations uint64
	// Number of total reclaims
	Reclaims uint64
	// Number of total frees
	Frees uint64
}

// ParseAVCStats returns the total SELinux access vector cache statistics,
// or error on failure.
func (fs FS) ParseAVCStats() (AVCStat, error) {
	avcStat := AVCStat{}

	file, err := os.Open(fs.selinux.Path("avc/cache_stats"))
	if err != nil {
		return avcStat, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Scan() // Skip header

	for scanner.Scan() {
		avcValues := strings.Fields(scanner.Text())

		if len(avcValues) != 6 {
			return avcStat, fmt.Errorf("invalid AVC stat line: %s",
				scanner.Text())
		}

		lookups, err := strconv.ParseUint(avcValues[0], 0, 64)
		if err != nil {
			return avcStat, fmt.Errorf("could not parse expected integer value for lookups")
		}

		hits, err := strconv.ParseUint(avcValues[1], 0, 64)
		if err != nil {
			return avcStat, fmt.Errorf("could not parse expected integer value for hits")
		}

		misses, err := strconv.ParseUint(avcValues[2], 0, 64)
		if err != nil {
			return avcStat, fmt.Errorf("could not parse expected integer value for misses")
		}

		allocations, err := strconv.ParseUint(avcValues[3], 0, 64)
		if err != nil {
			return avcStat, fmt.Errorf("could not parse expected integer value for allocations")
		}

		reclaims, err := strconv.ParseUint(avcValues[4], 0, 64)
		if err != nil {
			return avcStat, fmt.Errorf("could not parse expected integer value for reclaims")
		}

		frees, err := strconv.ParseUint(avcValues[5], 0, 64)
		if err != nil {
			return avcStat, fmt.Errorf("could not parse expected integer value for frees")
		}

		avcStat.Lookups += lookups
		avcStat.Hits += hits
		avcStat.Misses += misses
		avcStat.Allocations += allocations
		avcStat.Reclaims += reclaims
		avcStat.Frees += frees
	}

	return avcStat, scanner.Err()
}
