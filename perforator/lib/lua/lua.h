#pragma once

#include <contrib/libs/re2/re2/re2.h>
#include <llvm/Object/ObjectFile.h>
#include <perforator/lib/elf/elf.h>
#include <util/generic/maybe.h>
#include <util/generic/string.h>
#include <util/string/builder.h>

namespace NPerforator::NLinguist::NLua {

struct TLuaVersion {
    ui8 MajorVersion = 0;
    ui8 MinorVersion = 0;
};

struct TParsedLuaVersion {
    TLuaVersion Version;

    TString ToString() const {
        TStringBuilder builder;
        builder << ui64(Version.MajorVersion) << "." << ui64(Version.MinorVersion);
        return builder;
    }
};

class TLuaAnalyzer {
public:
    struct TLuaSymbols {
        // Fallback in case symbol size is not specified in symbol table of ELF
        static constexpr ui64 kFallbackLocationSize = 100;
        static constexpr ui64 kLjDispatchUpdateFallbackLocationSize = 1000;

        TMaybe<TStringBuf> LuaJitVersionSymbol; // Name of the symbol containing the LuaJIT version. Used to identify LuaJIT.

        TMaybe<NPerforator::NELF::TLocation> LuaClose;   // `lua_close`
        TMaybe<NPerforator::NELF::TLocation> LuaOpenJit; // `luaopen_jit`
        TMaybe<NPerforator::NELF::TLocation> LuaGc;      // `lua_gc`
    };

public:
    // Looking for `luaJIT_version_2_1_<number>` or `luaJIT_version_2_1_0_beta3` symbol.
    inline static const re2::RE2 kLuaJitVersionSymbolRegex = R"(^luaJIT_version_(\d+)_(\d+)_(.+)$)";
    // Looking for `LuaJIT 2.1.<number>` or `LuaJIT 2.1.0-beta3` literal.
    inline static const re2::RE2 kLuaJitVersionLiteralRegex = R"(^LuaJIT (\d+)\.(\d+)\.(.+)$)";

    // Symbol prefix to find suported LuaJIT binary
    static constexpr TStringBuf kLuaJitVersionSymbolPrefix = "luaJIT_version_";
    // Literal prefix to find supported LuaJIT binary
    static constexpr TStringBuf kLuaJitVersionLiteralPrefix = "LuaJIT ";

    // Used to identify LuaJIT binary and find some LuaJIT offsets
    static constexpr TStringBuf kLuaCloseSymbol = "lua_close";

    // Used to identify LuaJIT binary and find some LuaJIT offsets
    static constexpr TStringBuf kLuaOpenJitSymbol = "luaopen_jit";

    // Used to find some LuaJIT offsets
    static constexpr TStringBuf kLuaGcSymbol = "lua_gc";

public:
    explicit TLuaAnalyzer(const llvm::object::ObjectFile& file);

    TMaybe<TParsedLuaVersion> ParseVersion();

    // Offset from the start of global_State (G) structure to the `cur_L` field (L), which is the currently executing lua_State
    TMaybe<ui64> FindOffsetGToL();

    // Offset from the global_State (G) `g` field to the `dispatch` field of the GG_State (GG). See `GG_G2DISP` in LuaJIT
    TMaybe<ui64> FindOffsetGToDispatch();

    // Offset from the start of global_State (G) structure to the `vmstate` field, current execution state of this state
    TMaybe<ui64> FindOffsetGToVmState();

    // Size of the binary file. Used with base offset which is already available in BPF
    ui64 GetBinarySize();

    // Bounds of LuaJIT ASM VM. See algorithm description in `lua.cpp`.
    TMaybe<std::pair<ui64, ui64>> GetVMLocation();

private:
    void FindSymbolLocations();

    TMaybe<TLuaVersion> TryFindVersionInRodata();
    TMaybe<TLuaVersion> TryScanVersion(std::string_view data, const re2::RE2& regex);

private:
    const llvm::object::ObjectFile& File_;
    THolder<TLuaSymbols> Symbols_;
};

} // namespace NPerforator::NLinguist::NLua
