#pragma once

#include <llvm/BinaryFormat/Dwarf.h>
#include <llvm/DebugInfo/DWARF/DWARFExpression.h>

#include <util/system/types.h>
#include <util/generic/scope.h>

#include <optional>


namespace NPerforator::NBinaryProcessing::NUnwind {

struct TDwarfArgumentWildcard {
    ui64* Destination = nullptr;
};

inline TDwarfArgumentWildcard Wildcard(ui64* target) {
    return TDwarfArgumentWildcard{target};
}

struct TDwarfOperationPattern {
    llvm::dwarf::LocationAtom Opcode;
    std::optional<std::variant<ui64, TDwarfArgumentWildcard>> Arg;
};

class TDwarfExpressionPattern {
public:
    TDwarfExpressionPattern& Push(TDwarfOperationPattern pattern) {
        Pattern_.push_back(pattern);
        return *this;
    }

    TDwarfExpressionPattern& Push(llvm::dwarf::LocationAtom opcode) {
        return Push({opcode, std::nullopt});
    }

    TDwarfExpressionPattern& Push(llvm::dwarf::LocationAtom opcode, ui64 arg) {
        return Push({opcode, arg});
    }

    TDwarfExpressionPattern& Push(llvm::dwarf::LocationAtom opcode, TDwarfArgumentWildcard wildcard) {
        return Push({opcode, wildcard});
    }

    bool Matches(const llvm::DWARFExpression& expr) {
        auto iter = expr.begin();
        for (auto [code, arg] : Pattern_) {
            Y_DEFER {
                ++iter;
            };

            if (iter->getCode() != code) {
                return false;
            }
            if (!arg) {
                continue;
            }

            if (ui64* expected = std::get_if<ui64>(&arg.value()); expected && iter->getRawOperand(0) != *expected) {
                return false;
            }

            if (auto wildcard = std::get_if<TDwarfArgumentWildcard>(&arg.value())) {
                *wildcard->Destination = iter->getRawOperand(0);
            }
        }
        return iter == expr.end();
    }

private:
    llvm::SmallVector<TDwarfOperationPattern> Pattern_;
};

} // namespace NPerforator::NBinaryProcessing::NUnwind
