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

enum class ELuaVersionSource {
    LuaJitVersionSymbol,
    // Older LuaJIT versions (e.g. ones from debian/ubuntu distribution) do not
    // expose `luaJIT_version_*` symbol.
    // `lua_version` symbol is useless, as it always returns constant
    // value `LUA_VERSION_NUM` (501).
    // For now just hardcode 2.1.0 version if the binary happens to expose
    // `lua_close` and `luaopen_jit` or `luaopen_bit`, should be enough for
    // current purposes
    LuaJitDeduced,
};

struct TParsedLuaVersion {
    TLuaVersion Version;
    ELuaVersionSource Source;

    TString ToString() const {
        TStringBuilder builder;
        builder << ui64(Version.MajorVersion) << "."
                << ui64(Version.MinorVersion);
        return builder;
    }
};

class TLuaAnalyzer {
  public:
    struct TLuaSymbols {
        TMaybe<TStringBuf>
            LuaJitVersion; // Name of the symbol containing the LuaJIT version.
                           // Used as a main method to identify LuaJIT.

        TMaybe<NPerforator::NELF::TLocation> LuaClose;   // `lua_close`
        TMaybe<NPerforator::NELF::TLocation> LuaOpenJit; // `luaopen_jit`
        TMaybe<NPerforator::NELF::TLocation> LuaOpenBit; // `luaopen_bit`
    };

  public:
    // Main symbol prefix to find LuaJIT binary
    static constexpr TStringBuf kLuaJitVersionSymbolPrefix = "luaJIT_version_";
    static const re2::RE2 kLuaJitVersionRegex;

    // Used to identify LuaJIT binary and find some LuaJIT
    // offsets later
    static constexpr TStringBuf kLuaCloseSymbol = "lua_close";

    // Used to identify LuaJIT binary and find some LuaJIT
    // offsets later
    static constexpr TStringBuf kLuaOpenJitSymbol = "luaopen_jit";

    // Last hope to identify LuaJIT binary
    static constexpr TStringBuf kLuaOpenBitSymbol = "luaopen_bit";

  public:
    explicit TLuaAnalyzer(const llvm::object::ObjectFile &file);

    TMaybe<TParsedLuaVersion> ParseVersion();

    // offset from the start of global_State (G) structure to the `cur_L` field
    // (L), which is the currently executing lua_State
    TMaybe<ui64> ParseOffsetGtoL();

    // offset from the global_State (G) `g` field to the `dispatch` field of the
    // GG_State (GG)
    // See `GG_G2DISP` in LuaJIT
    TMaybe<ui64> ParseOffsetGtoDispatch();

    // Size of the binary file. Used with base offset which is already available
    // in BPF
    ui64 GetBinarySize();

    TMaybe<std::pair<ui64, ui64>> GetVMLocation();

  private:
    void ParseSymbolLocations();
    TMaybe<TLuaVersion> TryScanVersion(std::string_view data);

  private:
    const llvm::object::ObjectFile &File_;
    THolder<TLuaSymbols> Symbols_;
};

} // namespace NPerforator::NLinguist::NLua
