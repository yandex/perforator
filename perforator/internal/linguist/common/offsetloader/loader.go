// Package offsetloader provides generic version-keyed offset loading
// from embedded JSON files. Used by PHP and Python linguist agents.
package offsetloader

import (
	"embed"
	"fmt"
	"path"
	"regexp"
	"strings"
)

// VersionOffsets stores offsets keyed by language version with nearest-patch fallback.
type VersionOffsets[T any] struct {
	offsets map[LanguageVersion]T
}

// ParseFunc converts raw JSON bytes into a language-specific offset struct.
type ParseFunc[T any] func(data []byte) (T, error)

// Load reads all JSON files from an embed.FS directory, extracts version from
// the filename using the given regex (must have one capture group for the version string),
// and stores parsed offsets in a map.
//
// filenamePattern example: `php-(\d+\.\d+(?:\.\d+)?)-offsets\.json`
func Load[T any](fs embed.FS, dir string, filenamePattern *regexp.Regexp, parse ParseFunc[T]) *VersionOffsets[T] {
	vo := &VersionOffsets[T]{offsets: make(map[LanguageVersion]T)}

	entries, err := fs.ReadDir(dir)
	if err != nil {
		panic(fmt.Sprintf("offsetloader: failed to read directory %q: %v", dir, err))
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		matches := filenamePattern.FindStringSubmatch(entry.Name())
		if len(matches) < 2 {
			continue
		}

		versionKey, ok := NewLanguageVersionFromString(matches[1])
		if !ok {
			continue
		}

		jsonData, err := fs.ReadFile(path.Join(dir, entry.Name()))
		if err != nil {
			panic(fmt.Sprintf("offsetloader: failed to read %s: %v", entry.Name(), err))
		}

		offsets, err := parse(jsonData)
		if err != nil {
			panic(fmt.Sprintf("offsetloader: failed to parse %s: %v", entry.Name(), err))
		}

		vo.offsets[versionKey] = offsets
	}

	return vo
}

// Get returns offsets for the exact version, or falls back to the nearest
// patch release within the same major.minor.
func (vo *VersionOffsets[T]) Get(version LanguageVersion) (T, error) {
	if v, ok := vo.offsets[version]; ok {
		return v, nil
	}

	targetMajor, targetMinor := version.Major(), version.Minor()
	targetPatch := version.Patch()

	var best T
	bestDist := uint32(0xFFFFFFFF)
	found := false

	for key, offsets := range vo.offsets {
		if key.Major() != targetMajor || key.Minor() != targetMinor {
			continue
		}

		patch := key.Patch()
		dist := absDiff(targetPatch, patch)
		if dist < bestDist {
			bestDist = dist
			best = offsets
			found = true
		}
	}

	if found {
		return best, nil
	}

	var zero T
	return zero, fmt.Errorf("no offsets for version %d.%d.%d",
		version.Major(), version.Minor(), version.Patch())
}

// NewFromMap creates a VersionOffsets from an existing map. Useful for testing.
func NewFromMap[T any](m map[LanguageVersion]T) *VersionOffsets[T] {
	return &VersionOffsets[T]{offsets: m}
}

func absDiff(a, b uint32) uint32 {
	if a > b {
		return a - b
	}
	return b - a
}
