package unwindtable

import (
	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
)

type row struct {
	table *unwind.UnwindTable
	i     int
}

func (r *row) CFA() *unwind.UnwindRule {
	return r.table.Dict[r.table.Cfa[r.i]]
}

func (r *row) RBP() *unwind.UnwindRule {
	return r.table.Dict[r.table.Rbp[r.i]]
}

func (r *row) RA() *unwind.UnwindRule {
	return r.table.Dict[r.table.Ra[r.i]]
}

func (r *row) StartPC() uint64 {
	return r.table.StartPc[r.i]
}

func (r *row) PCRange() uint64 {
	return r.table.PcRange[r.i]
}
