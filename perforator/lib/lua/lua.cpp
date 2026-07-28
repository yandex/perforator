#include "lua.h"

#include <charconv>

#include <llvm/DebugInfo/DWARF/DWARFContext.h>
#include <llvm/DebugInfo/DWARF/DWARFDebugFrame.h>

#include <perforator/lib/lua/asm/decode.h>

namespace NPerforator::NLinguist::NLua {

TLuaAnalyzer::TLuaAnalyzer(const llvm::object::ObjectFile& file)
    : File_(file) {
}

TMaybe<TParsedLuaVersion> TLuaAnalyzer::ParseVersion() {
    FindSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    const auto& symbols = *Symbols_;

    // Main method. Find `luaJIT_version_*` (`LUAJIT_VERSION_SYM` macro) symbol and parse semver.
    if (symbols.LuaJitVersionSymbol) {
        if (const auto& version = TryScanVersion(*symbols.LuaJitVersionSymbol, kLuaJitVersionSymbolRegex)) {
            return MakeMaybe(TParsedLuaVersion {
                .Version = *version,
            });
        }
    }

    // Alternative method. Some binaries (e.g. in Debian) have `luaJIT_version_*` symbol removed.
    // 1. We need to make sure this is not a library but an interpreter. Binary must have `lua_close`.
    // 2. Then, to make sure this is LuaJIT and not Lua, the binary must have `luaopen_jit`.
    // 3. Last, we search for `LuaJIT 2.1.*` literal (`LUAJIT_VERSION` macro) in `.rodata` section.
    if (symbols.LuaClose && symbols.LuaOpenJit) {
        if (const auto& version = TryFindVersionInRodata()) {
            return MakeMaybe(TParsedLuaVersion {
                .Version = *version,
            });
        }
    }

    return Nothing();
}

TMaybe<ui64> TLuaAnalyzer::FindOffsetGToL() {
    FindSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    auto& luaCloseSymbol = Symbols_->LuaClose;
    if (!luaCloseSymbol || luaCloseSymbol->Address == 0) {
        // Binary doesn't have `lua_close`. It's not an interpreter.
        return Nothing();
    }

    if (luaCloseSymbol->Size == 0) {
        luaCloseSymbol->Size = TLuaSymbols::kFallbackLocationSize;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromSection(File_, *luaCloseSymbol, NELF::NSections::kText);
    if (!bytecode) {
        return Nothing();
    }

    return NAsm::DecodeLuaClose(File_.makeTriple(), luaCloseSymbol->Address, *bytecode);
}

TMaybe<ui64> TLuaAnalyzer::FindOffsetGToDispatch() {
    FindSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    auto& luaOpenJitSymbol = Symbols_->LuaOpenJit;
    if (!luaOpenJitSymbol || luaOpenJitSymbol->Address == 0) {
        // Binary doesn't have `luaopen_jit`. It's not LuaJIT.
        return Nothing();
    }

    if (luaOpenJitSymbol->Size == 0) {
        luaOpenJitSymbol->Size = TLuaSymbols::kFallbackLocationSize;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromSection(File_, *luaOpenJitSymbol, NELF::NSections::kText);
    if (!bytecode) {
        return Nothing();
    }

    TMaybe<ui64> ljDispatchUpdateAddress = NAsm::DecodeLuaOpenJit(File_.makeTriple(), luaOpenJitSymbol->Address, *bytecode);
    if (!ljDispatchUpdateAddress) {
        return Nothing();
    }

    NPerforator::NELF::TLocation ljDispatchUpdate = {
        .Address = luaOpenJitSymbol->Address + static_cast<i64>(*ljDispatchUpdateAddress),
        .Size = TLuaSymbols::kLjDispatchUpdateFallbackLocationSize,
    };

    bytecode = NPerforator::NELF::RetrieveContentFromSection(File_, ljDispatchUpdate, NELF::NSections::kText);
    if (!bytecode) {
        return Nothing();
    }

    return NAsm::DecodeLjDispatchUpdate(File_.makeTriple(), ljDispatchUpdate.Address, *bytecode);
}

TMaybe<ui64> TLuaAnalyzer::FindOffsetGToVmState() {
    FindSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    auto& luaGcSymbol = Symbols_->LuaGc;
    if (!luaGcSymbol || luaGcSymbol->Address == 0) {
        // Binary doesn't have `lua_gc`. That's weird.
        return Nothing();
    }

    if (luaGcSymbol->Size == 0) {
        luaGcSymbol->Size = TLuaSymbols::kFallbackLocationSize;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromSection(File_, *luaGcSymbol, NELF::NSections::kText);
    if (!bytecode) {
        return Nothing();
    }

    TMaybe<ui64> ljGcStepAddress = NAsm::DecodeLuaGc(File_.makeTriple(), luaGcSymbol->Address, *bytecode);
    if (!ljGcStepAddress) {
        return Nothing();
    }

    NPerforator::NELF::TLocation ljGcStep = {
        .Address = luaGcSymbol->Address + static_cast<i64>(*ljGcStepAddress),
        .Size = TLuaSymbols::kFallbackLocationSize,
    };

    bytecode = NPerforator::NELF::RetrieveContentFromSection(File_, ljGcStep, NELF::NSections::kText);
    if (!bytecode) {
        return Nothing();
    }

    return NAsm::DecodeLjGcStep(File_.makeTriple(), ljGcStep.Address, *bytecode);
}

ui64 TLuaAnalyzer::GetBinarySize() {
    return File_.getMemoryBufferRef().getBufferSize();
}

// To properly merge stacks and find if eBPF stopped inside LuaJIT VM (helps to get more relevant info), we need to find VM location.
// LuaJIT VM is a huge ASM blob. Function `emit_asm_debug` in `vm_x64.dasc` emits single `.debug_frame` and `.eh_frame` for the whole blob.
// This makes it easier to find VM location from the binary.
//
// First FDE is very big and has simple rule for CFA offset equal to `CFRAME_SIZE`.
// It covers `.Lbegin` (aka `lj_vm_asm_begin`) up to `lj_vm_ffi_call`.
//
// Second FDE is `lj_vm_ffi_call`, it needs special frame unwinding and hence it's placed at the end of the blob.
// Quote from `vm_x64.dasc`: "vm_ffi_call must be the last function in this object file!".
//
// This function searches for the first FDE with specific size and CFA offset.
// Then it searches for the `lj_vm_ffi_call` to find the actual end of VM.
// `lj_vm_ffi_call` might not be available if LuaJIT was built with `LUAJIT_DISABLE_FFI` option.
// This requires LuaJIT to be built without `LJ_NO_UNWIND` option.
TMaybe<std::pair<ui64, ui64>> TLuaAnalyzer::GetVMLocation() {
    // Analysis of builds from 2.1.0-beta3 to 7ff85518540617c0f97c4720558bb21c245994a8 (+ backported bitwise operators)
    // showed that VM size varies from 14'322 to 17'447.
    // We will expand and round bounds to be future-proof.
    static constexpr uint64_t kMinVMSize = 12'000;
    static constexpr uint64_t kMaxVMSize = 20'000;

    // CFRAME_SIZE on LJ_TARGET_X64
    static constexpr unsigned long kCFrameSize = 10 * 8;

    using FDE = const llvm::dwarf::FDE;

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

    auto getFdes = [&](const llvm::DWARFDebugFrame* ehFrame, auto predicate) {
        TVector<FDE*> result;
        for (const auto& entry : *ehFrame) {
            FDE* fde = llvm::dyn_cast<llvm::dwarf::FDE>(&entry);
            if (fde == nullptr) {
                Y_ENSURE(llvm::isa<llvm::dwarf::CIE>(&entry), "Unknown frame kind " << static_cast<int>(entry.getKind()));
                continue;
            }

            if (predicate(fde)) {
                result.emplace_back(fde);
            }
        }

        return result;
    };

    auto dwarfContext = llvm::DWARFContext::create(File_);
    const llvm::DWARFDebugFrame* ehFrame = Y_LLVM_RAISE(dwarfContext->getEHFrame());
    Y_ENSURE(ehFrame);

    // Search for "big asm blob"
    TVector<FDE*> vmFdeCandidates = getFdes(
        ehFrame,
        [&](FDE* fde) { return isFdeSizeInRange(fde) && checkFdeCfa(fde); }
    );
    if (vmFdeCandidates.empty()) {
        return Nothing();
    }
    std::ranges::sort(
        vmFdeCandidates,
        std::greater<> {},
        [](FDE* fde) { return fde->getAddressRange(); }
    );
    auto vmFde = vmFdeCandidates.front();

    uint64_t vmFdeStart = vmFde->getInitialLocation();
    uint64_t vmFdeEnd = vmFde->getInitialLocation() + vmFde->getAddressRange();

    // Search for "tail", aka `lj_vm_ffi_call`.
    TVector<FDE*> vmFfiCallFdeVector = getFdes(
        ehFrame,
        [&](FDE* fde) { return fde->getInitialLocation() == vmFdeEnd; }
    );
    if (!vmFfiCallFdeVector.empty()) {
        FDE* vmFfiCallFde = vmFfiCallFdeVector.front();
        vmFdeEnd = vmFfiCallFde->getInitialLocation() + vmFfiCallFde->getAddressRange();
    }

    // Final range:
    // If tail is found:     start of big FDE, end of `lj_vm_ffi_call`.
    // If tail is not found: start of big FDE, end of big FDE.
    return MakeMaybe(std::pair(vmFdeStart, vmFdeEnd));
}

void TLuaAnalyzer::FindSymbolLocations() {
    if (Symbols_) {
        return;
    }

    auto setSymbolIfFoundByPrefix = [&](const auto& symbols, const auto& symbolName, auto& target) {
        const auto& foundSymbol = std::ranges::find_if(
            symbols,
            [&](const auto& symbol) { return symbol.first.starts_with(symbolName); }
        );

        if (foundSymbol != symbols.end()) {
            target = std::move(foundSymbol->first);
        }
    };

    auto setSymbolIfFound = [&](const auto& symbols, const auto& symbolName, auto& target) {
        if (auto it = symbols.find(symbolName); it != symbols.end()) {
            target = std::move(it->second);
        }
    };

    auto startsWithPredicate = [](std::string_view prefix) {
        return [prefix](TStringBuf symbol) {
            return symbol.StartsWith(prefix);
        };
    };

    Symbols_ = MakeHolder<TLuaSymbols>();

    if (const auto& versionSymbols = NELF::RetrieveSymbols(File_, startsWithPredicate(kLuaJitVersionSymbolPrefix))) {
        setSymbolIfFoundByPrefix(*versionSymbols, kLuaJitVersionSymbolPrefix, Symbols_->LuaJitVersionSymbol);
    }

    if (const auto& symbols = NELF::RetrieveSymbols(File_, kLuaCloseSymbol, kLuaOpenJitSymbol, kLuaGcSymbol)) {
        setSymbolIfFound(*symbols, kLuaCloseSymbol, Symbols_->LuaClose);
        setSymbolIfFound(*symbols, kLuaOpenJitSymbol, Symbols_->LuaOpenJit);
        setSymbolIfFound(*symbols, kLuaGcSymbol, Symbols_->LuaGc);
    }
}

TMaybe<TLuaVersion> TLuaAnalyzer::TryFindVersionInRodata() {
    TMaybe<llvm::object::SectionRef> rodataSection = NELF::GetSection(File_, NPerforator::NELF::NSections::kRoData);
    if (!rodataSection) {
        return Nothing();
    }
    Y_LLVM_UNWRAP(content, rodataSection->getContents(), { return Nothing(); });

    size_t versionLiteralBegin = content.find(kLuaJitVersionLiteralPrefix);
    if (versionLiteralBegin == llvm::StringRef::npos) {
        return Nothing();
    }
    size_t versionEnd = content.find('\0', versionLiteralBegin);
    std::string_view versionLiteral = content.substr(versionLiteralBegin, versionEnd - versionLiteralBegin);

    return TryScanVersion(versionLiteral, kLuaJitVersionLiteralRegex);
}

TMaybe<TLuaVersion> TLuaAnalyzer::TryScanVersion(std::string_view input, const re2::RE2& regex) {
    std::string major, minor, patch;
    if (!re2::RE2::FindAndConsume(&input, regex, &major, &minor, &patch)) {
        return Nothing();
    }

    auto fromChars = [](const auto& string, auto& output) -> bool {
        return std::from_chars(string.data(), string.data() + string.size(), output).ec == std::errc {};
    };

    TLuaVersion luaVersion;

    if (!fromChars(major, luaVersion.MajorVersion)) {
        return Nothing();
    }

    if (!fromChars(minor, luaVersion.MinorVersion)) {
        return Nothing();
    }

    // Since 2090842410e0ba6f81fad310a77bf5432488249a commit LuaJIT has rolling release semantic.
    // This means patch version is a commit unix timestamp or `ROLLING`, when before it was `0-beta3`.
    // Currently we support LuaJIT since `2.1.0-beta3`.
    if (patch == "ROLLING" || patch == "0-beta3") {
        return luaVersion;
    }

    if (ui64 patchNumber = 0; fromChars(patch, patchNumber) && patchNumber != 0) {
        return luaVersion;
    }

    return Nothing();
}

} // namespace NPerforator::NLinguist::NLua
