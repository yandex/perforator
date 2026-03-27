package programstate

import (
	"github.com/cilium/ebpf"

	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/pkg/linux"
)

func (s *State) AddTLSConfig(id unwinder.BinaryId, tlsInfo *unwinder.TlsBinaryConfig) error {
	return s.maps.TlsStorage.Update(&id, tlsInfo, ebpf.UpdateAny)
}

func (s *State) DeleteTLSConfig(id unwinder.BinaryId) error {
	return s.maps.TlsStorage.Delete(&id)
}

func (s *State) AddPythonConfig(id unwinder.BinaryId, pythonInfo *unwinder.PythonConfig) error {
	return s.maps.PythonStorage.Update(id, pythonInfo, ebpf.UpdateAny)
}

func (s *State) DeletePythonConfig(id unwinder.BinaryId) error {
	return s.maps.PythonStorage.Delete(&id)
}

func (s *State) AddPhpConfig(id unwinder.BinaryId, phpInfo *unwinder.PhpConfig) error {
	return s.maps.PhpStorage.Update(id, phpInfo, ebpf.UpdateAny)

}

func (s *State) DeletePhpConfig(id unwinder.BinaryId) error {

	return s.maps.PhpStorage.Delete(&id)

}

func (s *State) AddPthreadConfig(id unwinder.BinaryId, pthreadInfo *unwinder.PthreadConfig) error {
	return s.maps.PthreadStorage.Update(id, pthreadInfo, ebpf.UpdateAny)
}

func (s *State) DeletePthreadConfig(id unwinder.BinaryId) error {
	return s.maps.PthreadStorage.Delete(&id)
}

// TODO: we can use batch lookups into bpf maps
func (s *State) SymbolizeInterpeter(key *unwinder.SymbolKey) (res unwinder.Symbol, exists bool) {
	err := s.maps.InterpreterSymbols.Lookup(key, &res)
	exists = (err == nil)
	return
}

func (s *State) PutJVMBinaryConfig(id unwinder.BinaryId, jvmConfig unwinder.JvmBinaryConfig) error {
	return s.maps.JvmBinaries.Put(id, jvmConfig)
}

func (s *State) DeleteJVMBinaryConfig(id unwinder.BinaryId) error {
	return s.maps.JvmBinaries.Delete(&id)
}

func (s *State) PutJVMProcessConfig(pid linux.CurrentNamespacePID, jvmConfig unwinder.JvmProcessConfig) error {
	return s.maps.JvmProcesses.Put(pid, jvmConfig)
}

func (s *State) DeleteJVMProcessConfig(pid linux.CurrentNamespacePID) error {
	return s.maps.JvmProcesses.Delete(&pid)
}
