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

// SELinux access vector cache hashtable statistics.
type AVCHashStat struct {
	// Number of entries
	Entries uint64
	// Number of buckets used
	BucketsUsed uint64
	// Number of buckets available
	BucketsAvailable uint64
	// Length of the longest chain
	LongestChain uint64
}

// ParseAVCHashStats returns the SELinux access vector cache hashtable
// statistics, or error on failure.
func (fs FS) ParseAVCHashStats() (AVCHashStat, error) {
	avcHashStat := AVCHashStat{}

	file, err := os.Open(fs.selinux.Path("avc/hash_stats"))
	if err != nil {
		return avcHashStat, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	scanner.Scan()
	entriesValue := strings.TrimPrefix(scanner.Text(), "entries: ")

	scanner.Scan()
	bucketsValues := strings.Split(scanner.Text(), "buckets used: ")
	bucketsValuesTuple := strings.Split(bucketsValues[1], "/")

	scanner.Scan()
	longestChainValue := strings.TrimPrefix(scanner.Text(), "longest chain: ")

	avcHashStat.Entries, err = strconv.ParseUint(entriesValue, 0, 64)
	if err != nil {
		return avcHashStat, fmt.Errorf("could not parse expected integer value for hash entries")
	}

	avcHashStat.BucketsUsed, err = strconv.ParseUint(bucketsValuesTuple[0], 0, 64)
	if err != nil {
		return avcHashStat, fmt.Errorf("could not parse expected integer value for hash buckets used")
	}

	avcHashStat.BucketsAvailable, err = strconv.ParseUint(bucketsValuesTuple[1], 0, 64)
	if err != nil {
		return avcHashStat, fmt.Errorf("could not parse expected integer value for hash buckets available")
	}

	avcHashStat.LongestChain, err = strconv.ParseUint(longestChainValue, 0, 64)
	if err != nil {
		return avcHashStat, fmt.Errorf("could not parse expected integer value for hash longest chain")
	}

	return avcHashStat, scanner.Err()
}
