package jvmregistry

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
	"github.com/yandex/perforator/perforator/proto/jvmsupp"
)

func synthesizeDWARF(methods []*jvmsupp.MethodInfo) (*unwind.UnwindTable, error) {
	t := &unwind.UnwindTable{}
	methods = slices.Clone(methods)
	slices.SortFunc(methods, func(a *jvmsupp.MethodInfo, b *jvmsupp.MethodInfo) int {
		return cmp.Compare(a.CodeBegin, b.CodeBegin)
	})
	for _, method := range methods {
		var ruleFP, rulePC, ruleSP *unwind.UnwindRule
		if !method.IsJit {
			continue
		}
		sz := int32(method.FrameSizeBytes)
		if int64(sz) != method.FrameSizeBytes {
			return nil, fmt.Errorf("frame size is too big for method %q", method.Name)
		}
		if sz < 0 {
			return nil, fmt.Errorf("negative frame size for method %q: %d", method.Name, sz)
		}
		ruleFP = &unwind.UnwindRule{
			Kind: &unwind.UnwindRule_CfaPlusOffset{
				CfaPlusOffset: &unwind.UnwindRule_CFAPlusOffset{
					Offset: -16,
				},
			},
		}
		rulePC = &unwind.UnwindRule{
			Kind: &unwind.UnwindRule_CfaPlusOffset{
				CfaPlusOffset: &unwind.UnwindRule_CFAPlusOffset{
					Offset: -8,
				},
			},
		}
		ruleSP = &unwind.UnwindRule{
			Kind: &unwind.UnwindRule_CfaPlusOffset{
				CfaPlusOffset: &unwind.UnwindRule_CFAPlusOffset{
					Offset: sz,
				},
			},
		}
		baseIdx := uint32(len(t.Dict))
		t.Dict = append(t.Dict, ruleFP)
		t.Dict = append(t.Dict, rulePC)
		t.Dict = append(t.Dict, ruleSP)

		t.Rbp = append(t.Rbp, baseIdx)
		t.Ra = append(t.Ra, baseIdx+1)
		t.Cfa = append(t.Cfa, baseIdx+2)
		t.StartPc = append(t.StartPc, method.CodeBegin)
		t.PcRange = append(t.PcRange, method.CodeEnd-method.CodeBegin)
	}
	return t, nil
}
