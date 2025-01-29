#include "gsym_symbolizer.h"

#include <perforator/lib/llvmex/llvm_exception.h>

#include <llvm/DebugInfo/Symbolize/Symbolize.h>

#include <fmt/format.h>

#include <algorithm>

namespace NPerforator::NGsym {

namespace {

std::string GetSourceFile(llvm::StringRef dir, llvm::StringRef base) {
    if (!dir.empty()) {
        if (base.empty()) {
            return std::string{dir};
        } else {
            return fmt::format("{}/{}", dir, base);
        }
    } else if (!base.empty()) {
        return std::string{base};
    }

    return {};
}

void PopulateDILineInfo(const llvm::gsym::SourceLocation& src, llvm::DILineInfo& dst) {
    dst.FileName = GetSourceFile(src.Dir, src.Base);
    dst.FunctionName = std::string{src.Name};
    dst.Line = src.Line;
}

}

TSymbolizer::TSymbolizer(std::string_view gsymPath)
    : llvm::gsym::GsymReader{Y_LLVM_RAISE(llvm::gsym::GsymReader::openFile(gsymPath))}
{
    const auto numAddresses = getNumAddresses();

    FunctionsState_.resize(numAddresses);
    Functions_.resize(numAddresses);
}

TSmallVector<llvm::DILineInfo> TSymbolizer::Symbolize(ui64 addr) {
    addr += getHeader().BaseAddress;

    auto idxOrError = getAddressIndex(addr);
    if (!idxOrError) {
        return {};
    }
    const auto idx = *idxOrError;

    if (!EnsureFunctionInfoAtIdx(idx)) {
        return {};
    }
    const auto& function = *Functions_[idx];

    if (!function.Range.contains(addr)) {
        // Apparently, some symbols could have zero size, and such symbols are considered a match.
        // https://github.com/llvm/llvm-project/blob/release/18.x/llvm/lib/DebugInfo/GSYM/GsymReader.cpp#L284
        if (!function.Range.empty()) {
            return {};
        }
    }

    auto stack = Symbolize(function, addr);
    std::reverse(stack.begin(), stack.end());

    TSmallVector<llvm::DILineInfo> result(stack.size());
    for (std::size_t i = 0; i < stack.size(); ++i) {
        PopulateDILineInfo(stack[i], result[i]);
    }

    return result;
}

bool TSymbolizer::EnsureFunctionInfoAtIdx(ui64 idx) {
    if (FunctionsState_.size() <= idx) {
        FunctionsState_.resize(idx);
        Functions_.resize(idx);
    }

    if (FunctionsState_[idx] == FunctionState::kNone) {
        auto functionInfoOrErr = getFunctionInfoAtIndex(idx);
        if (!functionInfoOrErr) {
            FunctionsState_[idx] = FunctionState::kFailed;
        } else {
            Functions_[idx].emplace(std::move(*functionInfoOrErr));
            FunctionsState_[idx] = FunctionState::kOk;
        }
    }

    return FunctionsState_[idx] == FunctionState::kOk;
}

TSmallVector<llvm::gsym::SourceLocation> TSymbolizer::Symbolize(
    const llvm::gsym::FunctionInfo& functionInfo,
    ui64 addr
) {
    llvm::gsym::SourceLocation topLevelEntry{};
    topLevelEntry.Name = getString(functionInfo.Name);

    if (!functionInfo.OptLineTable.has_value()) {
        return {topLevelEntry};
    }

    TSmallVector<llvm::gsym::SourceLocation> result;
    if (functionInfo.Inline.has_value()) {
        const auto inlineStackOpt = functionInfo.Inline->getInlineStack(addr);
        if (inlineStackOpt.has_value()) {
            ProcessInlineStack(*inlineStackOpt, result);
        }
    }
    result.push_back(topLevelEntry);

    ProcessTopLevelEntry(*functionInfo.OptLineTable, result.back(), addr);

    // Entries in the inlineStack have their Dir/Base pointing to "to where" they were inlined, not
    // "from where". Thus the last entry in inline stack gives us Dir/Base/Line of the top-level
    // function, and the first one is described by the lineEntry of the address at hand.
    // Cyclically shift the entries to restore the order.
    //
    // N.B. result is guaranteed to contain at least one entry at this point.

    // A B C -> A B C A -> A A B C -> C A B
    result.push_back(result.front());
    for (std::size_t i = result.size() - 1; i > 0; --i) {
        result[i].Dir = result[i - 1].Dir;
        result[i].Base = result[i - 1].Base;
        result[i].Line = result[i - 1].Line;
    }
    result.front() = result.back();
    result.pop_back();

    return result;
}

void TSymbolizer::ProcessInlineStack(
    const llvm::gsym::InlineInfo::InlineArray& inlineStack,
    TSmallVector<llvm::gsym::SourceLocation>& sourceLocations
) const {
    // The last entry in the inlineStack is the top-level function, which we have other means to construct.
    for (std::size_t i = 0; i + 1 < inlineStack.size(); ++i) {
        const auto& frame = *inlineStack[i];

        const auto fileNameOpt = getFile(frame.CallFile);
        if (!fileNameOpt.has_value()) {
            continue;
        }
        const auto& fileName = *fileNameOpt;

        sourceLocations.push_back(llvm::gsym::SourceLocation{
            .Name = getString(frame.Name),
            .Dir = getString(fileName.Dir),
            .Base = getString(fileName.Base),
            .Line = frame.CallLine,
        });
    }
}

void TSymbolizer::ProcessTopLevelEntry(
    const llvm::gsym::LineTable& lineTable,
    llvm::gsym::SourceLocation& topLevelEntry,
    ui64 addr
) const {
    // Find the last address which is <= our address and _assume_ that it is the entry we are looking for.
    // We might want to add some verification for that here, but for now it's fine.
    auto it = std::lower_bound(lineTable.begin(), lineTable.end(), addr,
        [](const auto& lineEntry, ui64 addr) {
        return lineEntry.Addr <= addr;
    });
    if (it != lineTable.begin()) {
        it = std::prev(it);
        if (it->Addr > addr) {
            return;
        }
    }

    const auto fileEntryOpt = getFile(it->File);
    if (!fileEntryOpt) {
        return;
    }
    const auto& fileEntry = *fileEntryOpt;

    topLevelEntry.Dir = getString(fileEntry.Dir);
    topLevelEntry.Base = getString(fileEntry.Base);
    topLevelEntry.Line = it->Line;
}

}
