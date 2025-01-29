#pragma once

#include <llvm/DebugInfo/GSYM/GsymReader.h>
#include <llvm/DebugInfo/Symbolize/Symbolize.h>

#include <util/generic/noncopyable.h>

#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <optional>
#include <string_view>
#include <vector>

namespace NPerforator::NGsym {

template<typename T>
using TSmallVector = NYT::TCompactVector<T, 4>;

class TSymbolizer final : llvm::gsym::GsymReader {
public:
    TSymbolizer(std::string_view gsymPath);

    TSmallVector<llvm::DILineInfo> Symbolize(ui64 addr);

private:
    bool EnsureFunctionInfoAtIdx(ui64 idx);

    TSmallVector<llvm::gsym::SourceLocation> Symbolize(
        const llvm::gsym::FunctionInfo& functionInfo,
        ui64 addr
    );

    void ProcessInlineStack(
        const llvm::gsym::InlineInfo::InlineArray& inlineStack,
        TSmallVector<llvm::gsym::SourceLocation>& sourceLocations
    ) const;
    void ProcessTopLevelEntry(
        const llvm::gsym::LineTable& lineTable,
        llvm::gsym::SourceLocation& topLevelEntry,
        ui64 addr
    ) const;

    enum class FunctionState {
        kNone, kOk, kFailed
    };
    std::vector<FunctionState> FunctionsState_;
    std::vector<std::optional<llvm::gsym::FunctionInfo>> Functions_;
};

}
