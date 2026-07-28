#include "lua.h"

#include <charconv>

#include <contrib/libs/re2/re2/stringpiece.h>

#include <llvm/ADT/AddressRanges.h>
#include <llvm/DebugInfo/DWARF/DWARFContext.h>
#include <llvm/DebugInfo/DWARF/DWARFDebugFrame.h>
#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>

#include <perforator/lib/elf/elf.h>
#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>
#include <perforator/lib/lua/asm/decode.h>

#include <util/generic/adaptor.h>
#include <util/generic/array_ref.h>
#include <util/generic/vector.h>
#include <util/stream/format.h>
#include <util/string/builder.h>

namespace NPerforator::NLinguist::NLua {

TLuaAnalyzer::TLuaAnalyzer(const llvm::object::ObjectFile& file)
    : File_(file) {}

TMaybe<TParsedLuaVersion> TLuaAnalyzer::ParseVersion() {
    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    const auto& symbols = *Symbols_;

    // Main method. Find `luaJIT_version_*` symbol and parse semver.
    if (symbols.LuaJitVersion) {
        if (const auto& version = TryScanVersion(*symbols.LuaJitVersion)) {
            return MakeMaybe(TParsedLuaVersion{
                .Version = *version,
            });
        }
    }

    // Alternative method.
    // 1. We need to make sure this is not a library but an interpreter. Binary
    // must have `lua_close`
    // 2. Then, to make sure this is LuaJIT and not Lua, the binary must have
    // `luaopen_jit`
    if (symbols.LuaClose && symbols.LuaOpenJit) {
        // TODO: find "LuaJIT 2.1.***" string inside the binary
        // Return hard-coded version
        return MakeMaybe(TParsedLuaVersion{
            .Version = {.MajorVersion = 2, .MinorVersion = 1},
        });
    }

    return Nothing();
}

TMaybe<ui64> TLuaAnalyzer::ParseOffsetGtoL() {
    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    auto& luaCloseSymbol = Symbols_->LuaClose;
    if (!luaCloseSymbol || luaCloseSymbol->Address == 0) {
        // Binary doesn't have `lua_close`. It's not an interpreter.
        return Nothing();
    }

    if (luaCloseSymbol->Size == 0) {
        // Fallback in case symbol size is not specified in symbol table of ELF
        luaCloseSymbol->Size = TLuaSymbols::kFallbackLocationSize;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromSection(
        File_, *luaCloseSymbol, NELF::NSections::kText);
    if (!bytecode) {
        return Nothing();
    }

    return NAsm::DecodeLuaClose(
        File_.makeTriple(), luaCloseSymbol->Address, *bytecode);
}

TMaybe<ui64> TLuaAnalyzer::ParseOffsetGtoDispatch() {
    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    auto& luaOpenJitSymbol = Symbols_->LuaOpenJit;
    if (!luaOpenJitSymbol || luaOpenJitSymbol->Address == 0) {
        return Nothing();
    }

    if (luaOpenJitSymbol->Size == 0) {
        luaOpenJitSymbol->Size = TLuaSymbols::kFallbackLocationSize;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromSection(
        File_, *luaOpenJitSymbol, NELF::NSections::kText);
    if (!bytecode) {
        return Nothing();
    }

    TMaybe<ui64> ljDispatchUpdateAddress = NAsm::DecodeLuaOpenJit(
        File_.makeTriple(), luaOpenJitSymbol->Address, *bytecode);
    if (!ljDispatchUpdateAddress) {
        return Nothing();
    }

    NPerforator::NELF::TLocation ljDispatchUpdate = {
        .Address = luaOpenJitSymbol->Address +
                   static_cast<i64>(*ljDispatchUpdateAddress),
        .Size = TLuaSymbols::kLjDispatchUpdateFallbackLocationSize,
    };

    bytecode = NPerforator::NELF::RetrieveContentFromSection(
        File_, ljDispatchUpdate, NELF::NSections::kText);
    if (!bytecode) {
        return Nothing();
    }

    return NAsm::DecodeLjDispatchUpdate(
        File_.makeTriple(), ljDispatchUpdate.Address, *bytecode);
}

ui64 TLuaAnalyzer::GetBinarySize() {
    return File_.getMemoryBufferRef().getBufferSize();
}

// emit_asm_debug
// TODO: Explain what we are getting here and why
TMaybe<std::pair<ui64, ui64>> TLuaAnalyzer::GetVMLocation() {
    // Analysis of builds from 2.1.0-beta3 to
    // 7ff85518540617c0f97c4720558bb21c245994a8 (+ backported bitwise operators)
    // showed that VM size varies from 14'322 to 17'447.
    // We will expand and round bounds to be future-proof.
    static constexpr uint64_t kMinVMSize = 12'000;
    static constexpr uint64_t kMaxVMSize = 20'000;

    // CFRAME_SIZE on LJ_TARGET_X64
    static constexpr unsigned long kCFrameSize = 10 * 8;

    auto dwarfContext = llvm::DWARFContext::create(File_);
    const llvm::DWARFDebugFrame* ehFrame =
        Y_LLVM_RAISE(dwarfContext->getEHFrame());
    Y_ENSURE(ehFrame);

    using FDE = const llvm::dwarf::FDE;
    TVector<FDE*> candidates;

    auto isFdeSizeInRange = [](FDE* fde) {
        auto fdeSize = fde->getAddressRange();
        return kMinVMSize <= fdeSize && fdeSize < kMaxVMSize;
    };

    auto checkFdeCfa = [](FDE* fde) {
        for (const auto& cfi : fde->cfis()) {
            if (cfi.Opcode == llvm::dwarf::DW_CFA_def_cfa_offset) {
                return cfi.Ops[0] == kCFrameSize;
            }
        }

        return false;
    };

    for (const auto& entry : ehFrame->entries()) {
        FDE* fde = llvm::dyn_cast<llvm::dwarf::FDE>(&entry);
        if (fde == nullptr) {
            Y_ENSURE(llvm::isa<llvm::dwarf::CIE>(&entry),
                "Unknown frame kind " << (int)entry.getKind());
            continue;
        }

        if (isFdeSizeInRange(fde) && checkFdeCfa(fde)) {
            candidates.emplace_back(fde);
        }
    }

    if (candidates.empty()) {
        return Nothing();
    }

    std::ranges::sort(candidates, std::greater<>{},
        [](FDE* fde) { return fde->getAddressRange(); });

    // TODO: get vm_ffi_call
    auto vmFde = candidates.front();

    return MakeMaybe(std::pair(vmFde->getInitialLocation(),
        vmFde->getInitialLocation() + vmFde->getAddressRange()));
}

void TLuaAnalyzer::ParseSymbolLocations() {
    if (Symbols_) {
        return;
    }

    auto setSymbolIfFoundByPrefix = [&](const auto& symbols,
                                        const auto& symbolName, auto& target) {
        const auto& foundSymbol =
            std::ranges::find_if(symbols, [&](const auto& symbol) {
                return symbol.first.starts_with(symbolName);
            });

        if (foundSymbol != symbols.end()) {
            target = std::move(foundSymbol->first);
        }
    };

    auto setSymbolIfFound = [&](const auto& symbols, const auto& symbolName,
                                auto& target) {
        if (auto it = symbols.find(symbolName); it != symbols.end()) {
            target = std::move(it->second);
        }
    };

    Symbols_ = MakeHolder<TLuaSymbols>();

    auto startsWithLuaJitVersion = [](TStringBuf symbol){
        return symbol.StartsWith(kLuaJitVersionSymbolPrefix);
    };

    if (const auto& versionSymbols =
            NELF::RetrieveSymbols(File_, startsWithLuaJitVersion)) {
        setSymbolIfFoundByPrefix(*versionSymbols, kLuaJitVersionSymbolPrefix,
            Symbols_->LuaJitVersion);
    }

    if (const auto& symbols =
            NELF::RetrieveSymbols(File_, kLuaCloseSymbol, kLuaOpenJitSymbol)) {
        setSymbolIfFound(*symbols, kLuaCloseSymbol, Symbols_->LuaClose);
        setSymbolIfFound(*symbols, kLuaOpenJitSymbol, Symbols_->LuaOpenJit);
    }
}

TMaybe<TLuaVersion> TLuaAnalyzer::TryScanVersion(std::string_view input) {
    std::string major, minor;

    if (!re2::RE2::FindAndConsume(
            &input, kLuaJitVersionRegex, &major, &minor)) {
        return Nothing();
    }

    auto fromChars = [](const auto& string, auto& output) -> bool {
        return std::from_chars(
                   string.data(), string.data() + string.size(), output)
                   .ec == std::errc{};
    };

    TLuaVersion luaVersion;

    if (!fromChars(major, luaVersion.MajorVersion)) {
        return Nothing();
    }

    if (!fromChars(minor, luaVersion.MinorVersion)) {
        return Nothing();
    }

    return luaVersion;
}

}  // namespace NPerforator::NLinguist::NLua
