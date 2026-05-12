#include "lua.h"

#include <charconv>
#include <ranges>

#include <contrib/libs/re2/re2/stringpiece.h>

#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>

#include <perforator/lib/elf/elf.h>
#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>
#include <perforator/lib/lua/asm/x86-64/decode.h>

#include <util/generic/adaptor.h>
#include <util/generic/array_ref.h>
#include <util/generic/vector.h>
#include <util/stream/format.h>
#include <util/string/builder.h>

#define WORK_IN_PROGRESS_LOG(...)                                              \
    fprintf(stderr, __VA_ARGS__);                                              \
    fflush(stderr);

namespace NPerforator::NLinguist::NLua {

const re2::RE2
    TLuaAnalyzer::kLuaJitVersionRegex(R"(^luaJIT_version_(\d+)_(\d+)_(\d+))");

TLuaAnalyzer::TLuaAnalyzer(const llvm::object::ObjectFile &file)
    : File_(file) {}

TMaybe<TParsedLuaVersion> TLuaAnalyzer::ParseVersion() {
    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    auto &&symbols = *Symbols_;

    WORK_IN_PROGRESS_LOG("SSE4: ParseVersion\n");

    // Main method. Find `luaJIT_version_*` symbol and parse semver.
    if (symbols.LuaJitVersion) {
        if (auto &&version = TryScanVersion(*symbols.LuaJitVersion)) {
            WORK_IN_PROGRESS_LOG("SSE4: ParseVersion has LuaJitVersion\n");

            return MakeMaybe(TParsedLuaVersion{
                .Version = *version,
                .Source = ELuaVersionSource::LuaJitVersionSymbol});
        }
    }

    // Alternative method.
    // 1. We need to make sure this is not a library but an interpreter. Binary
    // must have `lua_close`
    // 2. Then, to make sure this is LuaJIT and not Lua, the binary must have
    // `luaopen_jit` or `luaopen_bit` (in case JIT is disabled)
    if (symbols.LuaClose && (symbols.LuaOpenJit || symbols.LuaOpenBit)) {
        WORK_IN_PROGRESS_LOG(
            "SSE4: ParseVersion has LuaClose and LuaOpenJit or LuaOpen\n");

        // Return hard-coded version
        return MakeMaybe(
            TParsedLuaVersion{.Version =
                                  {
                                      .MajorVersion = 2,
                                      .MinorVersion = 1,
                                      .MicroVersion = 0,
                                  },
                              .Source = ELuaVersionSource::LuaJitDeduced});
    }

    WORK_IN_PROGRESS_LOG("SSE4: ParseVersion has nothing\n");

    return Nothing();
}

// see opentelemetry-ebpf-profiler, function findOffsetsFromLuaClose,
// interpreter/luajit/extractor_x86.go
TMaybe<ui64> TLuaAnalyzer::ParseOffsetGtoL() {
    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    if (!Symbols_->LuaClose || Symbols_->LuaClose->Address == 0) {
        // Binary doesn't have `lua_close`. It's not an interpreter.
        return Nothing();
    }

    auto &&LuaCloseSymbol = *Symbols_->LuaClose;

    if (LuaCloseSymbol.Size == 0) {
        // fallback in case symbol size is not specified in symbol table of ELF
        LuaCloseSymbol.Size = 100;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromTextSection(
        File_, LuaCloseSymbol);
    if (!bytecode) {
        return Nothing();
    }

    return NAsm::NX86::DecodeLuaClose(File_.makeTriple(), *bytecode);
}

// See opentelemetry-ebpf-profiler, function findLjDispatchUpdateAddr,
// interpreter/luajit/extractor_x86.go
TMaybe<ui64> TLuaAnalyzer::ParseOffsetGtoDispatch() {
    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    if (!Symbols_->LuaOpenJit || Symbols_->LuaOpenJit->Address == 0) {
        return Nothing();
    }

    NPerforator::NELF::TLocation &LuaOpenJitSymbol = *Symbols_->LuaOpenJit;

    if (LuaOpenJitSymbol.Size == 0) {
        // fallback in case symbol size is not specified in symbol table of ELF
        LuaOpenJitSymbol.Size = 100;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromTextSection(
        File_, LuaOpenJitSymbol);
    if (!bytecode) {
        return Nothing();
    }

    TMaybe<ui64> lj_dispatch_update =
        NAsm::NX86::DecodeLuaOpenJit(File_.makeTriple(), *bytecode);
    if (!lj_dispatch_update) {
        return Nothing();
    }

    NPerforator::NELF::TLocation LjDispatchUpdate = {
        .Address =
            LuaOpenJitSymbol.Address + static_cast<i64>(*lj_dispatch_update),
        .Size = 1000};

    bytecode = NPerforator::NELF::RetrieveContentFromTextSection(
        File_, LjDispatchUpdate);
    if (!bytecode) {
        return Nothing();
    }

    return NAsm::NX86::DecodeLjDispatchUpdate(File_.makeTriple(), *bytecode);
}

ui64 TLuaAnalyzer::GetBinarySize() {
    return File_.getMemoryBufferRef().getBufferSize();
}

void TLuaAnalyzer::ParseSymbolLocations() {
    if (Symbols_) {
        return;
    }

    auto setSymbolIfFoundByPrefix = [&](auto &&symbols, auto &&symbolName,
                                        auto &target) {
        auto &&found_symbol = std::ranges::find_if(symbols, [&](auto &&symbol) {
            return symbol.first.starts_with(symbolName);
        });

        if (found_symbol != symbols.end()) {
            target = std::move(found_symbol->first);
        }
    };

    auto setSymbolIfFound = [&](auto &&symbols, auto &&symbolName,
                                auto &target) {
        if (auto it = symbols.find(symbolName); it != symbols.end()) {
            target = std::move(it->second);
        }
    };

    Symbols_ = MakeHolder<TLuaSymbols>();

    if (auto &&version_symbols =
            NELF::RetrieveSymbolsByPrefix(File_, kLuaJitVersionSymbolPrefix)) {
        setSymbolIfFoundByPrefix(*version_symbols, kLuaJitVersionSymbolPrefix,
                                 Symbols_->LuaJitVersion);
    }

    if (auto &&symbols = NELF::RetrieveSymbols(
            File_, kLuaCloseSymbol, kLuaOpenJitSymbol, kLuaOpenBitSymbol)) {
        setSymbolIfFound(*symbols, kLuaCloseSymbol, Symbols_->LuaClose);
        setSymbolIfFound(*symbols, kLuaOpenJitSymbol, Symbols_->LuaOpenJit);
        setSymbolIfFound(*symbols, kLuaOpenBitSymbol, Symbols_->LuaOpenBit);
    }
}

TMaybe<TLuaVersion> TLuaAnalyzer::TryScanVersion(std::string_view input) {
    std::string major, minor, micro;

    if (!re2::RE2::FindAndConsume(&input, kLuaJitVersionRegex, &major, &minor,
                                  &micro)) {
        return Nothing();
    }

    auto from_chars = [](auto &&string, auto &&output) {
        return std::from_chars(string.data(), string.data() + string.size(),
                               output)
                   .ec == std::errc{};
    };

    TLuaVersion luaVersion;

    if (!from_chars(major, luaVersion.MajorVersion)) {
        return Nothing();
    }

    if (!from_chars(minor, luaVersion.MinorVersion)) {
        return Nothing();
    }

    if (!from_chars(micro, luaVersion.MicroVersion)) {
        return Nothing();
    }

    return luaVersion;
}

} // namespace NPerforator::NLinguist::NLua
