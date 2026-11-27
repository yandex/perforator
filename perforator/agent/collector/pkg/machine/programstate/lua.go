package programstate

import (
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

type LuaJITVM struct {
	Process     linux.ProcessKey
	GlobalState uint64
	Config      unwinder.LuaConfig
	BinaryBase  uint64
}

func (s *State) LuaJITVMs() ([]LuaJITVM, error) {
	var result []LuaJITVM
	var key unwinder.LuaJitVmKey
	var value unwinder.LuaJitVm
	iterator := s.maps.LuaJitVms.Iterate()
	for iterator.Next(&key, &value) {
		result = append(result, LuaJITVM{
			Process:     linux.ProcessKey{Pid: linux.CurrentNamespacePID(key.Pid), ProcessStartTime: key.ProcessStarttime},
			GlobalState: key.GlobalState,
			Config:      value.Config,
			BinaryBase:  value.BinaryBase,
		})
	}
	return result, iterator.Err()
}

func (s *State) DeleteLuaJITVM(vm LuaJITVM) error {
	key := unwinder.LuaJitVmKey{
		Pid:              uint32(vm.Process.Pid),
		ProcessStarttime: vm.Process.ProcessStartTime,
		GlobalState:      vm.GlobalState,
	}
	return s.maps.LuaJitVms.Delete(&key)
}
