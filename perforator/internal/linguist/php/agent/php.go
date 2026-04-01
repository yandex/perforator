package agent

import (
	"fmt"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/php"
	"github.com/yandex/perforator/perforator/internal/linguist/common/offsetloader"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

// IsVersionSupported checks whether offsets are available for this PHP version.
func IsVersionSupported(version *php.PhpVersion) bool {
	if version == nil {
		return false
	}
	_, err := GetOffsets(version)
	return err == nil
}

// ParsePhpUnwinderConfig converts a PhpConfig proto to an unwinder config
// with version-appropriate struct offsets.
func ParsePhpUnwinderConfig(config *php.PhpConfig) (*unwinder.PhpConfig, error) {
	offsets, err := GetOffsets(config.Version)
	if err != nil {
		return nil, fmt.Errorf("unsupported PHP version %d.%d.%d: %w",
			config.Version.Major, config.Version.Minor, config.Version.Release, err)
	}

	return &unwinder.PhpConfig{
		Version:                 offsetloader.NewLanguageVersion(config.Version.Major, config.Version.Minor, config.Version.Release).EncodeToUint32(),
		ExecutorGlobalsElfVaddr: config.ExecutorGlobalsELFVaddr,
		Offsets:                 *offsets,
	}, nil
}
