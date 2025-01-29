#include "dwarf_matcher.h"
#include "ehframe.h"

#include "rule_dict.h"

#include <perforator/agent/preprocessing/proto/parse/parse.pb.h>
#include <perforator/lib/llvmex/llvm_exception.h>
#include <perforator/lib/permutation/permutation.h>
#include <perforator/lib/tls/parser/tls.h>

#include <library/cpp/iterator/enumerate.h>
#include <library/cpp/streams/zstd/zstd.h>

#include <util/generic/algorithm.h>
#include <util/generic/maybe.h>
#include <util/generic/xrange.h>
#include <util/generic/vector.h>
#include <util/generic/yexception.h>

#include <llvm/DebugInfo/DWARF/DWARFContext.h>
#include <llvm/DebugInfo/DWARF/DWARFDebugFrame.h>
#include <llvm/MC/MCRegisterInfo.h>
#include <llvm/MC/TargetRegistry.h>
#include <llvm/Object/ObjectFile.h>
#include <llvm/Support/TargetSelect.h>


namespace NPerforator::NBinaryProcessing::NUnwind {

////////////////////////////////////////////////////////////////////////////////

void DifferentiateUnwindTable(NUnwind::UnwindTable& table) {
    // Delta-encode pc ranges
    ui64 pc = 0;
    for (int i : xrange(table.start_pc_size())) {
        ui64 start_pc = table.start_pc(i);
        ui64 pc_range = table.pc_range(i);

        Y_ENSURE(start_pc >= pc, "Mismatched pc: " << start_pc << " < " << pc);
        ui64 end = start_pc + pc_range;
        table.set_start_pc(i, start_pc - pc);
        pc = end;
    }
}

void IntegrateUnwindTable(NUnwind::UnwindTable& table) {
    ui64 pc = 0;
    for (int i : xrange(table.start_pc_size())) {
        table.set_start_pc(i, table.start_pc(i) + pc);
        pc = table.start_pc(i) + table.pc_range(i);
    }
}

////////////////////////////////////////////////////////////////////////////////

void DeltaEncode(NUnwind::UnwindTable& table) {
    MultiSort(
         *table.mutable_start_pc(),
         *table.mutable_pc_range(),
         *table.mutable_cfa(),
         *table.mutable_rbp(),
         *table.mutable_ra()
    );

    NUnwind::DifferentiateUnwindTable(table);
}


////////////////////////////////////////////////////////////////////////////////

// Ugly hack to make pointer to the private member.
// UnwindLocation::Dereference is private and there is no getter for it.
// See http://bloglitb.blogspot.com/2011/12/access-to-private-members-safer.html?m=1
template <typename Tag, typename Tag::type M>
struct Backdoor {
    friend typename Tag::type get(Tag) {
        return M;
    }
};

template <typename Tag, typename Member>
struct TagBase {
    typedef Member type;
    friend type get(Tag);
};

struct UnwindLocationTag : TagBase<UnwindLocationTag, bool llvm::dwarf::UnwindLocation::*> {};

template struct Backdoor<UnwindLocationTag, &llvm::dwarf::UnwindLocation::Dereference>;

bool IsDerefLocation(const llvm::dwarf::UnwindLocation& loc) {
    return loc.*get(UnwindLocationTag{});
}

////////////////////////////////////////////////////////////////////////////////

void FillRegisterLocation(UnwindRule* rule, const llvm::dwarf::UnwindLocation& loc) {
    rule->set_dereference(IsDerefLocation(loc));

    switch (loc.getLocation()) {
    case llvm::dwarf::UnwindLocation::Unspecified:
    case llvm::dwarf::UnwindLocation::Undefined:
    case llvm::dwarf::UnwindLocation::Same:
        // These are meaningless for unwinding
        rule->mutable_unsupported();
        break;
    case llvm::dwarf::UnwindLocation::CFAPlusOffset:
        if (loc.getOffset() == -8) {
            rule->mutable_cfa_minus8();
        } else {
            rule->mutable_cfa_plus_offset()->set_offset(loc.getOffset());
        }
        break;
    case llvm::dwarf::UnwindLocation::RegPlusOffset:
        rule->mutable_register_offset()->set_register_(loc.getRegister());
        rule->mutable_register_offset()->set_offset(loc.getOffset());
        break;
    case llvm::dwarf::UnwindLocation::DWARFExpr: {
        auto expr = loc.getDWARFExpressionBytes();
        // Tricky part: support some common dwarf expressions.

        {
            // 1. Static hand-written CFA rule for .plt sections:
            // DW_OP_breg7 RSP+8, DW_OP_breg16 RIP+0, DW_OP_lit15, DW_OP_and, DW_OP_lit10, DW_OP_ge, DW_OP_lit3, DW_OP_shl, DW_OP_plus

            bool matches = NPerforator::NBinaryProcessing::NUnwind::TDwarfExpressionPattern{}
                .Push(llvm::dwarf::DW_OP_breg7, 8)
                .Push(llvm::dwarf::DW_OP_breg16, 0)
                .Push(llvm::dwarf::DW_OP_lit15)
                .Push(llvm::dwarf::DW_OP_and)
                .Push(llvm::dwarf::DW_OP_lit10)
                .Push(llvm::dwarf::DW_OP_ge)
                .Push(llvm::dwarf::DW_OP_lit3)
                .Push(llvm::dwarf::DW_OP_shl)
                .Push(llvm::dwarf::DW_OP_plus)
                .Matches(*expr);

            if (matches) {
                rule->mutable_plt_section();
                break;
            }
        }

        {
            // 2. Static hand-written CFA rule from openssl:
            // DW_OP_breg[67] R[BS]P+544, DW_OP_deref, DW_OP_plus_uconst 0x8

            for (int regno : {6, 7}) {
                ui64 offset = 0;
                ui64 bias = 0;
                auto atom = static_cast<llvm::dwarf::LocationAtom>(llvm::dwarf::DW_OP_breg0 + regno);
                bool matches = NPerforator::NBinaryProcessing::NUnwind::TDwarfExpressionPattern{}
                    .Push(atom, NPerforator::NBinaryProcessing::NUnwind::Wildcard(&offset))
                    .Push(llvm::dwarf::DW_OP_deref)
                    .Push(llvm::dwarf::DW_OP_plus_uconst, NPerforator::NBinaryProcessing::NUnwind::Wildcard(&bias))
                    .Matches(*expr);

                if (matches) {
                    rule->mutable_register_deref_offset()->set_register_(regno);
                    rule->mutable_register_deref_offset()->set_offset(static_cast<i64>(offset));
                    rule->mutable_register_deref_offset()->set_bias(bias);
                    break;
                }
            }
        }

        {
            // 3. Static hand-written CFA rule from openssl:
            // DW_OP_breg7 RSP+8, DW_OP_breg9 R9+0, DW_OP_lit8, DW_OP_mul, DW_OP_plus, DW_OP_deref, DW_OP_plus_uconst 0x8

            bool matches = NPerforator::NBinaryProcessing::NUnwind::TDwarfExpressionPattern{}
                .Push(llvm::dwarf::DW_OP_breg7, 8)
                .Push(llvm::dwarf::DW_OP_breg9, 0)
                .Push(llvm::dwarf::DW_OP_lit8)
                .Push(llvm::dwarf::DW_OP_mul)
                .Push(llvm::dwarf::DW_OP_plus)
                .Push(llvm::dwarf::DW_OP_deref)
                .Push(llvm::dwarf::DW_OP_plus_uconst, 8)
                .Matches(*expr);

            if (matches) {
                // TODO
                rule->mutable_unsupported();
                break;
            }
        }
        break;
    }
    case llvm::dwarf::UnwindLocation::Constant:
        rule->mutable_constant()->set_value(loc.getConstant());
        break;
    // no explicit default to trigger compilation error on new location kind
    }
}

////////////////////////////////////////////////////////////////////////////////

void RemapRules(google::protobuf::RepeatedField<ui32>* rules, const NUnwind::TRuleDict& dict) {
    for (auto& rule : *rules) {
        rule = dict.RemapRule(rule);
    }
}

UnwindTable BuildUnwindTable(llvm::object::ObjectFile* objectFile) {
    auto dwarfContext = llvm::DWARFContext::create(*objectFile);

    bool isEh = true;
    const llvm::DWARFDebugFrame* ehFrame = nullptr;
    if (isEh) {
        ehFrame = Y_LLVM_RAISE(dwarfContext->getEHFrame());
    } else {
        ehFrame = Y_LLVM_RAISE(dwarfContext->getDebugFrame());
    }
    Y_ENSURE(ehFrame);

    NUnwind::TRuleDictBuilder dictBuilder;
    NPerforator::NBinaryProcessing::NUnwind::UnwindTable unwtable;

    for (auto&& frame : ehFrame->entries()) {
        const llvm::dwarf::FDE* fde = llvm::dyn_cast<llvm::dwarf::FDE>(&frame);
        if (fde == nullptr) {
            Y_ENSURE(llvm::isa<llvm::dwarf::CIE>(&frame), "Unknown eh frame kind " << (int)frame.getKind());
            continue;
        }

        const llvm::dwarf::CIE* cie = fde->getLinkedCIE();
        Y_ENSURE(cie, "Empty CIE for FDE at " << frame.getOffset());

        llvm::dwarf::UnwindTable table = Y_LLVM_RAISE(llvm::dwarf::UnwindTable::create(fde));

        for (const auto& [i, row] : Enumerate(table)) {
            auto pcAddressAndRange = [&]() -> TMaybe<std::pair<uint64_t, uint64_t>> {
                auto startAddress = row.getAddress();
                auto endAddress = [&]() -> uint64_t {
                    if (i + 1 < table.size()) {
                        return table[i + 1].getAddress();
                    }
                        return fde->getInitialLocation() + fde->getAddressRange();
                }();

                Y_ENSURE(endAddress >= startAddress);
                // We met DWARFs with fde->getInitialLocation() + fde->getAddressRange() == row.Address(),
                // however according to section 6.4.1 of DWARF spec it is incorrect.
                // Nonetheless, we still give best efforts to process these binaries.
                if (startAddress == endAddress) {
                    return Nothing();
                }
                return {{startAddress, endAddress - startAddress}};
            }();

            if (!pcAddressAndRange) {
                continue;
            }
            auto [pcAddress, pcRange] = *pcAddressAndRange;
            unwtable.add_start_pc(pcAddress);
            unwtable.add_pc_range(pcRange);

            NUnwind::UnwindRule cfa;
            NUnwind::UnwindRule rbp;
            NUnwind::UnwindRule ra;

            auto loc = row.getRegisterLocations().getRegisterLocation(cie->getReturnAddressRegister());
            if (loc) {
                ra.set_dereference(IsDerefLocation(*loc));

                switch (loc.value().getLocation()) {
                case llvm::dwarf::UnwindLocation::Unspecified:
                case llvm::dwarf::UnwindLocation::Undefined:
                case llvm::dwarf::UnwindLocation::Same:
                    // These are meaningless for unwinding
                    ra.mutable_unsupported();
                    break;
                case llvm::dwarf::UnwindLocation::CFAPlusOffset:
                    if (loc->getOffset() == -8) {
                        ra.mutable_cfa_minus8();
                    } else {
                        ra.mutable_cfa_plus_offset()->set_offset(loc->getOffset());
                    }
                    break;
                case llvm::dwarf::UnwindLocation::RegPlusOffset:
                    ra.mutable_register_offset()->set_register_(loc->getRegister());
                    ra.mutable_register_offset()->set_offset(loc->getOffset());
                    break;
                case llvm::dwarf::UnwindLocation::DWARFExpr:
                    // We do not support dwarf expressions here.
                    ra.mutable_unsupported();
                    break;
                case llvm::dwarf::UnwindLocation::Constant:
                    ra.mutable_constant()->set_value(loc->getConstant());
                    break;
                // no explicit default to trigger compilation error on new location kind
                }
            }

            // Fill CFA value.
            FillRegisterLocation(&cfa, row.getCFAValue());

            // Fill RBP value. It is needed in ~25% of unwind rules.
            if (auto loc = row.getRegisterLocations().getRegisterLocation(6/*rbp=*/)) {
                FillRegisterLocation(&rbp, *loc);
            } else {
                rbp.mutable_unsupported();
            }

            unwtable.add_cfa(dictBuilder.Add(std::move(cfa)));
            unwtable.add_rbp(dictBuilder.Add(std::move(rbp)));
            unwtable.add_ra(dictBuilder.Add(std::move(ra)));
        }
    }

    auto dict = std::move(dictBuilder).Finish();
    unwtable.mutable_dict()->Assign(dict.Rules().begin(), dict.Rules().end());
    RemapRules(unwtable.mutable_cfa(), dict);
    RemapRules(unwtable.mutable_rbp(), dict);
    RemapRules(unwtable.mutable_ra(), dict);

    // Sanity check.
    auto len = unwtable.start_pc_size();
    Y_ENSURE(len == unwtable.pc_range_size());
    Y_ENSURE(len == unwtable.cfa_size());
    Y_ENSURE(len == unwtable.rbp_size());
    Y_ENSURE(len == unwtable.ra_size());

    return unwtable;
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NBinaryProcessing::NUnwind
