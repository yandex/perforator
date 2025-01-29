package profiler

import "github.com/yandex/perforator/perforator/pkg/linux"

type EventListener interface {
	OnSampleStored(pid linux.ProcessID)
}
