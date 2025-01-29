package filter

import (
	"github.com/yandex/perforator/perforator/pkg/storage/profile/meta"
)

type MicroscopeType int

const (
	AbstractFilter MicroscopeType = iota
	PodFilter
	NodeFilter
	ServiceFilter
)

type Filter interface {
	// Return true if profile metadata satisfies any microscope
	Filter(meta *meta.ProfileMetadata) bool
}
