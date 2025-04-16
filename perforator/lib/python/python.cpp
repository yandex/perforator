#include "python.h"

#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>

#include <perforator/lib/elf/elf.h>
#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <util/generic/adaptor.h>
#include <util/generic/array_ref.h>
#include <util/generic/vector.h>
#include <util/stream/format.h>

#include <contrib/libs/re2/re2/stringpiece.h>
#include <util/string/builder.h>

namespace NPerforator::NLinguist::NPython {

TPythonAnalyzer::TPythonAnalyzer(const llvm::object::ObjectFile& file) : File_(file) {}

void TPythonAnalyzer::ParseSymbolLocations() {
    if (Symbols_) {
        return;
    }

    Symbols_ = MakeHolder<TSymbols>();

    auto dynamicSymbols = NELF::RetrieveDynamicSymbols(File_,
        kPyVersionSymbol,
        kPyThreadStateGetCurrentSymbol,
        kPyGetVersionSymbol,
        kPyRuntimeSymbol,
        kPyGILStateEnsureSymbol,
        kPyInterpreterStateHeadSymbol
    );

    auto setSymbolIfFound = [&](const THashMap<TStringBuf, NPerforator::NELF::TLocation>& symbols, TStringBuf symbolName, TMaybe<NELF::TLocation>& target) {
        if (auto it = symbols.find(symbolName); it != symbols.end()) {
            target = it->second;
        }
    };

    if (dynamicSymbols) {
        setSymbolIfFound(*dynamicSymbols, kPyVersionSymbol, Symbols_->PyVersion);
        setSymbolIfFound(*dynamicSymbols, kPyThreadStateGetCurrentSymbol, Symbols_->GetCurrentThreadState);
        setSymbolIfFound(*dynamicSymbols, kPyGetVersionSymbol, Symbols_->PyGetVersion);
        setSymbolIfFound(*dynamicSymbols, kPyRuntimeSymbol, Symbols_->PyRuntime);
        setSymbolIfFound(*dynamicSymbols, kPyGILStateEnsureSymbol, Symbols_->PyGILStateEnsure);
        setSymbolIfFound(*dynamicSymbols, kPyInterpreterStateHeadSymbol, Symbols_->PyInterpreterStateHead);
    }

    auto symbols = NELF::RetrieveSymbols(File_, kCurrentFastGetSymbol);
    if (symbols) {
        setSymbolIfFound(*symbols, kCurrentFastGetSymbol, Symbols_->CurrentFastGet);
    }

    return;
}

template <typename ELFT>
TMaybe<TPythonVersion> TryParseVersionFromPyVersionSymbol(
    const llvm::object::ObjectFile& file,
    const NPerforator::NELF::TLocation& pyVersion
) {
    if (pyVersion.Address == 0) {
        return Nothing();
    }

    if (pyVersion.Size != sizeof(ui32) && pyVersion.Size != sizeof(ui64)) {
        return Nothing();
    }

    auto content = NPerforator::NELF::RetrieveContentFromRodataSection(file, pyVersion);
    if (!content) {
        return Nothing();
    }

    TVector<ui8> versionBytes(content->begin(), content->end());
    if constexpr (ELFT::TargetEndianness == llvm::endianness::little) {
        Reverse(versionBytes.begin(), versionBytes.end());
    }

    ui32 skipBytes = (versionBytes.size() == sizeof(ui64)) ? sizeof(ui32) : 0;

    return MakeMaybe(TPythonVersion{
        .MajorVersion = ui8(versionBytes[skipBytes + 0]),
        .MinorVersion = ui8(versionBytes[skipBytes + 1]),
        .MicroVersion = ui8(versionBytes[skipBytes + 2]),
    });
}

TMaybe<TPythonVersion> TryScanVersion(
    TConstArrayRef<char> data
) {
    /*
     * Python version string formats:
     * - Python < 3.3.0: Can be either X.Y (e.g. "2.6") or X.Y.Z (e.g. "2.7.17")
     * - Python >= 3.3.0: Always X.Y.Z format (e.g. "3.3.0", "3.12.1")
     */
    re2::StringPiece input(data.data(), data.size());
    std::string major, minor, micro, suffix;


    while (re2::RE2::FindAndConsume(&input, kPythonVersionRegex, &major, &minor, &micro, &suffix)) {
        ui8 majorVersion = static_cast<ui8>(std::stoi(major));
        ui8 minorVersion = static_cast<ui8>(std::stoi(minor));
        ui8 microVersion = micro.empty() ? 0 : static_cast<ui8>(std::stoi(micro));

        // For X.Y format, only accept versions < 3.3.0
        if (micro.empty() && (majorVersion == 3 && minorVersion >= 3)) {
            continue;
        }

        return TPythonVersion{
            .MajorVersion = majorVersion,
            .MinorVersion = minorVersion,
            .MicroVersion = microVersion,
        };
    }

    return Nothing();
}

TMaybe<TPythonVersion> TryParsePyGetVersion(
    const llvm::object::ObjectFile& file,
    const NPerforator::NELF::TLocation& pyGetVersion
) {
    NPerforator::NELF::TLocation symbol = pyGetVersion;
    if (symbol.Size == 0) {
        // fallback in case symbol size is not specified in symbol table of ELF
        symbol.Size = 64;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromTextSection(file, symbol);
    if (!bytecode) {
        return Nothing();
    }

    auto versionAddress = NAsm::NX86::DecodePyGetVersion(file.makeTriple(), symbol.Address, *bytecode);
    if (!versionAddress) {
        return Nothing();
    }

    NPerforator::NELF::TLocation pyVersionLocation{
        .Address = *versionAddress,
        .Size = 10
    };
    auto pyVersionContent = NPerforator::NELF::RetrieveContentFromRodataSection(file, pyVersionLocation);
    if (!pyVersionContent) {
        return Nothing();
    }

    return TryScanVersion(
        TConstArrayRef<char>(
            reinterpret_cast<const char*>(pyVersionContent->data()),
            pyVersionContent->size()
        )
    );
}

template <typename ELFT>
TMaybe<TParsedPythonVersion> ParseVersion(
    const llvm::object::ObjectFile& file,
    const TPythonAnalyzer::TSymbols& symbols
) {
    const llvm::object::ELFObjectFile<ELFT>* elf = llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(&file);
    if (!elf) {
        return Nothing();
    }

    // First try Py_Version symbol if available
    if (symbols.PyVersion) {
        if (auto version = TryParseVersionFromPyVersionSymbol<ELFT>(file, *symbols.PyVersion)) {
            return MakeMaybe(TParsedPythonVersion{
                .Version = *version,
                .Source = EPythonVersionSource::PyVersionSymbol
            });
        }
    }

    // Try to find PY_VERSION string through Py_GetVersion disassembly
    if (symbols.PyGetVersion) {
        if (auto version = TryParsePyGetVersion(*elf, *symbols.PyGetVersion)) {
            return MakeMaybe(TParsedPythonVersion{
                .Version = *version,
                .Source = EPythonVersionSource::PyGetVersionDisassembly
            });
        }
    }

    return Nothing();
}

TMaybe<TParsedPythonVersion> TPythonAnalyzer::ParseVersion() {
    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    #define TRY_ELF_TYPE(ELFT) \
    if (auto res = NPerforator::NLinguist::NPython::ParseVersion<ELFT>(File_, *Symbols_.Get())) { \
        return res; \
    }

    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)

#undef TRY_ELF_TYPE
    return Nothing();
}

TMaybe<NAsm::ThreadImageOffsetType> TPythonAnalyzer::ParseTLSPyThreadState() {
    if (File_.getArch() != llvm::Triple::x86 && File_.getArch() != llvm::Triple::x86_64) {
        return Nothing();
    }

    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    if (!Symbols_->GetCurrentThreadState) {
        return Nothing();
    }

    // current_fast_get might not be inlined into GetCurrentThreadState, so we should disassemble it instead of PyThreadState_GetCurrent.
    NPerforator::NELF::TLocation& getter = Symbols_->CurrentFastGet ? *Symbols_->CurrentFastGet : *Symbols_->GetCurrentThreadState;
    if (getter.Address == 0) {
        return Nothing();
    }
    if (getter.Size == 0) {
        // fallback in case symbol size is not specified in symbol table of ELF
        getter.Size = 100;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromTextSection(File_, getter);
    if (!bytecode) {
        return Nothing();
    }

    return Symbols_->CurrentFastGet ?
        NAsm::NX86::DecodeCurrentFastGet(File_.makeTriple(), *bytecode) :
        NAsm::NX86::DecodePyThreadStateGetCurrent(File_.makeTriple(), *bytecode);
}

bool IsPythonBinary(const llvm::object::ObjectFile& file) {
    auto dynamicSymbols = NELF::RetrieveDynamicSymbols(file, kPyGetVersionSymbol);
    // Also check that the address is not null, because symbols can be imported from dynamic libraries
    return (dynamicSymbols && dynamicSymbols->size() == 1 && (*dynamicSymbols)[kPyGetVersionSymbol].Address != 0);
}

TMaybe<ui64> TPythonAnalyzer::ParsePyRuntimeAddress() {
    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    if (!Symbols_->PyRuntime || Symbols_->PyRuntime->Address == 0) {
        return Nothing();
    }

    return MakeMaybe(Symbols_->PyRuntime->Address);
}

TMaybe<ui64> TPythonAnalyzer::ParseAutoTSSKeyAddress() {
    if (File_.getArch() != llvm::Triple::x86 && File_.getArch() != llvm::Triple::x86_64) {
        return Nothing();
    }

    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    if (!Symbols_->PyGILStateEnsure || Symbols_->PyGILStateEnsure->Address == 0) {
        return Nothing();
    }

    NPerforator::NELF::TLocation& pyGILStateEnsureSymbol = *Symbols_->PyGILStateEnsure;
    if (pyGILStateEnsureSymbol.Size == 0) {
        // fallback in case symbol size is not specified in symbol table of ELF
        pyGILStateEnsureSymbol.Size = 100;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromTextSection(File_, pyGILStateEnsureSymbol);
    if (!bytecode) {
        return Nothing();
    }

    return NAsm::NX86::DecodeAutoTSSKeyAddress(
        File_.makeTriple(),
        pyGILStateEnsureSymbol.Address,
        *bytecode
    );
}

TMaybe<ui64> TPythonAnalyzer::ParseInterpHeadAddress() {
    if (File_.getArch() != llvm::Triple::x86 && File_.getArch() != llvm::Triple::x86_64) {
        return Nothing();
    }

    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    if (!Symbols_->PyInterpreterStateHead || Symbols_->PyInterpreterStateHead->Address == 0) {
        return Nothing();
    }

    NPerforator::NELF::TLocation& pyInterpreterStateHeadSymbol = *Symbols_->PyInterpreterStateHead;
    if (pyInterpreterStateHeadSymbol.Size == 0) {
        pyInterpreterStateHeadSymbol.Size = 30;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromTextSection(File_, pyInterpreterStateHeadSymbol);
    if (!bytecode) {
        return Nothing();
    }

    return NAsm::NX86::DecodeInterpHeadAddress(File_.makeTriple(), pyInterpreterStateHeadSymbol.Address, *bytecode);
}

} // namespace NPerforator::NLinguist::NPython
