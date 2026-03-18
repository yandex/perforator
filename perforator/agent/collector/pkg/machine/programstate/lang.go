package programstate

import (
	"github.com/cilium/ebpf"

	"github.com/yandex/perforator/perforator/internal/unwinder"
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

func (s *State) SymbolizeInterpeter(key *unwinder.SymbolKey) (res unwinder.Symbol, exists bool) {
	err := s.maps.InterpreterSymbols.Lookup(key, &res)
	exists = (err == nil)
	return
}

// SymbolizeInterpreterBatch resolves n symbol keys in a single sequential scan,
// eliminating per-frame syscall round-trips: O(n) → 1 kernel transition per batch.
// found[i] == false iff key[i] is absent from the eBPF LRU hash map.
func (s *State) SymbolizeInterpreterBatch(keys []unwinder.SymbolKey) (symbols []unwinder.Symbol, found []bool) {
	symbols = make([]unwinder.Symbol, len(keys))
	found = make([]bool, len(keys))
	for i := range keys {
		found[i] = s.maps.InterpreterSymbols.Lookup(&keys[i], &symbols[i]) == nil
	}
	return
}
