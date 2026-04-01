package offsetloader

import (
	"fmt"
	"strings"
)

type LanguageVersion struct {
	major uint32
	minor uint32
	patch uint32
}

func NewLanguageVersion(major, minor, patch uint32) LanguageVersion {
	return LanguageVersion{
		major: major,
		minor: minor,
		patch: patch,
	}
}

func (v LanguageVersion) Major() uint32 { return v.major }
func (v LanguageVersion) Minor() uint32 { return v.minor }
func (v LanguageVersion) Patch() uint32 { return v.patch }

func (v LanguageVersion) EncodeToUint32() uint32 {
	return (uint32(v.major) << 16) | (uint32(v.minor) << 8) | uint32(v.patch)
}

func NewLanguageVersionFromString(version string) (LanguageVersion, bool) {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return LanguageVersion{}, false
	}

	var major, minor, patch uint32
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return LanguageVersion{}, false
	}
	if _, err := fmt.Sscanf(parts[1], "%d", &minor); err != nil {
		return LanguageVersion{}, false
	}
	if len(parts) >= 3 {
		if _, err := fmt.Sscanf(parts[2], "%d", &patch); err != nil {
			return LanguageVersion{}, false
		}
	}

	return NewLanguageVersion(major, minor, patch), true
}
