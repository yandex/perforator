package process

import "github.com/yandex/perforator/perforator/pkg/linux"

type ProcessInfo interface {
	ProcessID() linux.ProcessID
	// returned map may not be modified
	Env() map[string]string
}

type Listener interface {
	OnProcessDiscovery(info ProcessInfo)
	OnProcessDeath(pid linux.ProcessID)
}
