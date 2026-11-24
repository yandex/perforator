package unwindtable

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/yandex/perforator/perforator/agent/preprocessing/proto/unwind"
	"github.com/yandex/perforator/perforator/internal/unwinder"
)

func protoDerefCfaRule(register uint32, offset, bias int32) *unwind.UnwindRule {
	return &unwind.UnwindRule{Kind: &unwind.UnwindRule_RegisterDerefOffset_{RegisterDerefOffset: &unwind.UnwindRule_RegisterDerefOffset{
		Register: register,
		Offset:   offset,
		Bias:     bias,
	}}}
}

func unsupportedRule() *unwind.UnwindRule {
	return &unwind.UnwindRule{Kind: &unwind.UnwindRule_Unsupported_{Unsupported: &unwind.UnwindRule_Unsupported{}}}
}

func testRow(cfa, rbp *unwind.UnwindRule) row {
	return row{
		table: &unwind.UnwindTable{
			Dict: []*unwind.UnwindRule{
				cfa,
				{Kind: &unwind.UnwindRule_CfaMinus8{CfaMinus8: &unwind.UnwindRule_CFAMinus8{}}},
				rbp,
			},
			StartPc: []uint64{0x1010},
			PcRange: []uint64{0x20},
			Cfa:     []uint32{0},
			Ra:      []uint32{1},
			Rbp:     []uint32{2},
		},
	}
}

func TestFillRuleDerefCfa(t *testing.T) {
	for _, tc := range []struct {
		name     string
		register uint32
		offset   int32
		bias     int32
		expected unwinder.CfaUnwindRule
	}{
		{
			name:     "openssl",
			register: uint32(unwinder.DwarfCfiStackRegisterNumber),
			offset:   152,
			bias:     8,
			expected: unwinder.CfaUnwindRule{
				Kind:   unwinder.UnwindRuleRegisterDerefOffsetPlus8,
				Regno:  uint8(unwinder.DwarfCfiStackRegisterNumber),
				Offset: 152,
			},
		},
		{
			name:     "gcc-drap",
			register: 6,
			offset:   -8,
			bias:     0,
			expected: unwinder.CfaUnwindRule{
				Kind:   unwinder.UnwindRuleRegisterDerefOffset,
				Regno:  6,
				Offset: -8,
			},
		},
		{
			name:     "unsupported-bias",
			register: uint32(unwinder.DwarfCfiStackRegisterNumber),
			offset:   16,
			bias:     16,
			expected: unwinder.CfaUnwindRule{
				Kind:   unwinder.UnwindRuleUnsupported,
				Regno:  uint8(unwinder.DwarfCfiStackRegisterNumber),
				Offset: 16,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := &unwinder.UnwindTablePageLeaf{}
			fillRule(testRow(protoDerefCfaRule(tc.register, tc.offset, tc.bias), unsupportedRule()), page, 3, 0x1000)

			require.Equal(t, tc.expected, page.Rules[3].Cfa)
			require.Equal(t, int8(-8), page.Rules[3].Ra.Offset)
			require.Equal(t, uint32(0x10), page.Pc[3])
			require.Equal(t, uint32(0x20), page.Ranges[3])
		})
	}
}

func TestFillRuleRbpSavedAtRbp(t *testing.T) {
	cfa := protoDerefCfaRule(6, -8, 0)
	rbpAt := func(register uint32, offset int32, dereference bool) *unwind.UnwindRule {
		return &unwind.UnwindRule{
			Kind: &unwind.UnwindRule_RegisterOffset{RegisterOffset: &unwind.UnwindRule_RegisterPlusOffset{
				Register: register,
				Offset:   offset,
			}},
			Dereference: dereference,
		}
	}

	for _, tc := range []struct {
		name     string
		rbp      *unwind.UnwindRule
		expected int8
	}{
		{name: "gcc-drap", rbp: rbpAt(6, 0, true), expected: int8(unwinder.DwarfUnwindRbpRuleAtRbp)},
		{name: "nonzero-offset", rbp: rbpAt(6, 8, true), expected: dwarfUnwindRuleUndefined},
		{name: "other-register", rbp: rbpAt(7, 0, true), expected: dwarfUnwindRuleUndefined},
		{name: "not-dereferenced", rbp: rbpAt(6, 0, false), expected: dwarfUnwindRuleUndefined},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page := &unwinder.UnwindTablePageLeaf{}
			fillRule(testRow(cfa, tc.rbp), page, 0, 0x1000)
			require.Equal(t, tc.expected, page.Rules[0].Rbp.Offset)
		})
	}
}

func TestFillRuleRegisterCfa(t *testing.T) {
	// openssl aesni_cbc_encrypt: CFA = r11 + 8. The eBPF unwinder resolves registers other than rsp and rbp.
	cfa := &unwind.UnwindRule{Kind: &unwind.UnwindRule_RegisterOffset{RegisterOffset: &unwind.UnwindRule_RegisterPlusOffset{
		Register: 11,
		Offset:   8,
	}}}

	page := &unwinder.UnwindTablePageLeaf{}
	fillRule(testRow(cfa, unsupportedRule()), page, 0, 0x1000)
	require.Equal(t, unwinder.CfaUnwindRule{Kind: unwinder.UnwindRuleRegisterOffset, Regno: 11, Offset: 8}, page.Rules[0].Cfa)
}
