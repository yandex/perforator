package jvmregistry

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
	"github.com/yandex/perforator/perforator/internal/unwinder"
	"github.com/yandex/perforator/perforator/proto/jvmsupp"
)

func synthesizeDWARF(methods []*jvmsupp.MethodInfo) (*unwind.UnwindTable, error) {
	t := &unwind.UnwindTable{}
	methods = slices.Clone(methods)
	slices.SortFunc(methods, func(a *jvmsupp.MethodInfo, b *jvmsupp.MethodInfo) int {
		return cmp.Compare(a.CodeBegin, b.CodeBegin)
	})

	var pushRule = func(begin, end uint64, ruleFP, rulePC, ruleSP *unwind.UnwindRule) {
		baseIdx := uint32(len(t.Dict))
		t.Dict = append(t.Dict, ruleFP)
		t.Dict = append(t.Dict, rulePC)
		t.Dict = append(t.Dict, ruleSP)

		t.Rbp = append(t.Rbp, baseIdx)
		t.Ra = append(t.Ra, baseIdx+1)
		t.Cfa = append(t.Cfa, baseIdx+2)
		t.StartPc = append(t.StartPc, begin)
		t.PcRange = append(t.PcRange, end-begin)
	}

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
		mainRuleStart := method.CodeBegin
		if method.FrameCompleteOffset > 0 {
			if method.CodeBegin+uint64(method.FrameCompleteOffset) >= method.CodeEnd {
				return nil, fmt.Errorf("frame complete offset %d is too big for method %q with frame size %d", method.FrameCompleteOffset, method.Name, sz)
			}

			mainRuleStart += uint64(method.FrameCompleteOffset)
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
						Offset: 8,
					},
				},
			}

			pushRule(method.CodeBegin, mainRuleStart, ruleFP, rulePC, ruleSP)
		}
		var fpOffset int32 = -16
		if sz == 0 {
			fpOffset = int32(unwinder.DwarfUnwindCfaRuleUndefined)
		}
		ruleFP = &unwind.UnwindRule{
			Kind: &unwind.UnwindRule_CfaPlusOffset{
				CfaPlusOffset: &unwind.UnwindRule_CFAPlusOffset{
					Offset: fpOffset,
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
		cfaOffset := sz
		if sz == 0 {
			cfaOffset = 8
		}
		ruleSP = &unwind.UnwindRule{
			Kind: &unwind.UnwindRule_CfaPlusOffset{
				CfaPlusOffset: &unwind.UnwindRule_CFAPlusOffset{
					Offset: cfaOffset,
				},
			},
		}
		pushRule(mainRuleStart, method.CodeEnd, ruleFP, rulePC, ruleSP)
	}
	return t, nil
}
