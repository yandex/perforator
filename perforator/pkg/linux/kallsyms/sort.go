package kallsyms

import "sort"

type symbolSort struct {
	addresses []uint64
	symbols   []string
	modules   []string
}

func (s *symbolSort) Len() int {
	return len(s.addresses)
}

func (s *symbolSort) Swap(i, j int) {
	s.addresses[i], s.addresses[j] = s.addresses[j], s.addresses[i]
	s.symbols[i], s.symbols[j] = s.symbols[j], s.symbols[i]
	s.modules[i], s.modules[j] = s.modules[j], s.modules[i]
}

func (s *symbolSort) Less(i, j int) bool {
	return s.addresses[i] < s.addresses[j]
}

func sortKallsyms(addresses []uint64, symbols, modules []string) {
	sort.Sort(&symbolSort{addresses, symbols, modules})
}
