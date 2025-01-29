#include "python.h"

#include <llvm/MC/MCContext.h>
#include <llvm/MC/MCTargetOptions.h>
#include <llvm/MC/TargetRegistry.h>
#include <llvm/Object/ELF.h>
#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>

#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <util/generic/adaptor.h>
#include <util/generic/array_ref.h>
#include <util/generic/vector.h>
#include <util/stream/format.h>

#include <contrib/libs/re2/re2/stringpiece.h>
#include <util/string/builder.h>

namespace NPerforator::NLinguist::NPython {

// Some limit for decoding of instructions of `_PyThreadState_GetCurrent` function
constexpr ui64 kMaxPyThreadStateGetCurrentBytecodeLength = 64;

TPythonAnalyzer::TPythonAnalyzer(llvm::object::ObjectFile* file) : File_(file) {}

template <typename ELFT>
THolder<TPythonAnalyzer::TGlobalsAddresses> ParseGlobalsAddresses(llvm::object::ObjectFile* file) {
    llvm::object::ELFObjectFile<ELFT>* elf = llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(file);
    if (!elf) {
        return nullptr;
    }

    THolder<TPythonAnalyzer::TGlobalsAddresses> res = MakeHolder<TPythonAnalyzer::TGlobalsAddresses>();

    for (auto&& symbol : elf->getDynamicSymbolIterators()) {
        Y_LLVM_UNWRAP(name, symbol.getName(), { continue; });
        Y_LLVM_UNWRAP(address, symbol.getAddress(), { continue; });

        if (TStringBuf{name.data(), name.size()} == kPyVersionSymbol) {
            res->PyVersionAddress = address;
        }

        if (TStringBuf{name.data(), name.size()} == kPyThreadStateGetCurrentSymbol) {
            res->GetCurrentThreadStateAddress = address;
        }

        if (TStringBuf{name.data(), name.size()} == kPyGetVersionSymbol) {
            res->PyGetVersionAddress = address;
        }

        if (TStringBuf{name.data(), name.size()} == kPyRuntimeSymbol) {
            res->PyRuntimeAddress = address;
        }
    }

    for (auto&& symbol : elf->symbols()) {
        Y_LLVM_UNWRAP(name, symbol.getName(), { continue; });
        Y_LLVM_UNWRAP(address, symbol.getAddress(), { continue; });

        if (TStringBuf{name.data(), name.size()} == kCurrentFastGetSymbol) {
            res->CurrentFastGetAddress = address;
        }
    }

    return res;
}

void TPythonAnalyzer::ParseGlobalsAddresses() {
    if (GlobalsAddresses_ != nullptr) {
        return;
    }

    #define TRY_ELF_TYPE(ELFT) \
    if (auto res = NPerforator::NLinguist::NPython::ParseGlobalsAddresses<ELFT>(File_)) { \
        GlobalsAddresses_ = std::move(res); \
        return; \
    }

    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)

#undef TRY_ELF_TYPE
    return;
}

template <typename ELFT>
TMaybe<llvm::object::SectionRef> LookForSection(
    llvm::object::ELFObjectFile<ELFT>* elf,
    TStringBuf name
) {
    for (auto&& section : elf->sections()) {
        Y_LLVM_UNWRAP(sectionName, section.getName(), { continue; });

        if (TStringBuf{sectionName.data(), sectionName.size()} == name) {
            return section;
        }
    }

    return Nothing();
}

template <typename ELFT>
TMaybe<TPythonVersion> TryParseVersionFromPyVersionSymbol(
    const llvm::object::SectionRef& section,
    llvm::StringRef sectionData,
    ui64 pyVersionAddress
) {
    if (pyVersionAddress < section.getAddress()) {
        return Nothing();
    }

    ui64 offset = pyVersionAddress - section.getAddress();
    if (offset + sizeof(ui32) >= sectionData.size()) {
        return Nothing();
    }

    TStringBuf versionView{sectionData.data() + offset, sizeof(ui32)};
    TVector<char> versionBytes(versionView.begin(), versionView.end());
    if constexpr (ELFT::TargetEndianness == llvm::endianness::little) {
        Reverse(versionBytes.begin(), versionBytes.end());
    }

    return MakeMaybe(TPythonVersion{
        .MajorVersion = ui8(versionBytes[0]),
        .MinorVersion = ui8(versionBytes[1]),
        .MicroVersion = ui8(versionBytes[2]),
    });
}

template <typename ELFT>
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

template <typename ELFT>
TMaybe<TPythonVersion> TryParsePyGetVersion(
    llvm::object::ELFObjectFile<ELFT>* elf,
    ui64 pyGetVersionAddress
) {
    auto textSection = LookForSection(elf, kTextSectionName);
    if (!textSection) {
        return Nothing();
    }

    Y_LLVM_UNWRAP(sectionData, textSection->getContents(), { return Nothing(); });
    if (pyGetVersionAddress < textSection->getAddress()) {
        return Nothing();
    }

    ui64 offset = pyGetVersionAddress - textSection->getAddress();
    if (offset >= sectionData.size()) {
        return Nothing();
    }

    TConstArrayRef<ui8> bytecode(
        reinterpret_cast<const ui8*>(sectionData.data()) + offset,
        Min<size_t>(64, sectionData.size() - offset)  // Limit to first 64 bytes
    );

    auto versionAddress = NDecode::NX86::DecodePyGetVersion(elf, pyGetVersionAddress, bytecode);
    if (!versionAddress) {
        return Nothing();
    }

    auto rodataSection = LookForSection(elf, kRoDataSectionName);
    if (!rodataSection) {
        return Nothing();
    }

    Y_LLVM_UNWRAP(rodataData, rodataSection->getContents(), { return Nothing(); });
    if (*versionAddress < rodataSection->getAddress()) {
        return Nothing();
    }

    ui64 versionOffset = *versionAddress - rodataSection->getAddress();
    if (versionOffset >= rodataData.size()) {
        return Nothing();
    }

    return TryScanVersion<ELFT>(TConstArrayRef<char>(
        rodataData.data() + versionOffset,
        Min<size_t>(10, rodataData.size() - versionOffset)  // Limit to 10 bytes which is enough for "X.YY.ZZZ"
    ));
}

template <typename ELFT>
TMaybe<TParsedPythonVersion> ParseVersion(
    llvm::object::ObjectFile* file,
    const TPythonAnalyzer::TGlobalsAddresses& addresses
) {
    llvm::object::ELFObjectFile<ELFT>* elf = llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(file);
    if (!elf) {
        return Nothing();
    }

    // First try Py_Version symbol if available
    if (addresses.PyVersionAddress != 0) {
        auto section = LookForSection(elf, kRoDataSectionName);
        if (section) {
            Y_LLVM_UNWRAP(sectionData, section->getContents(), { return Nothing(); });
            if (auto version = TryParseVersionFromPyVersionSymbol<ELFT>(*section, sectionData, addresses.PyVersionAddress)) {
                return MakeMaybe(TParsedPythonVersion{
                    .Version = *version,
                    .Source = EPythonVersionSource::PyVersionSymbol
                });
            }
        }
    }

    // Try to find PY_VERSION string through Py_GetVersion disassembly
    if (addresses.PyGetVersionAddress != 0) {
        if (auto version = TryParsePyGetVersion(elf, addresses.PyGetVersionAddress)) {
            return MakeMaybe(TParsedPythonVersion{
                .Version = *version,
                .Source = EPythonVersionSource::PyGetVersionDisassembly
            });
        }
    }

    return Nothing();
}

TMaybe<TParsedPythonVersion> TPythonAnalyzer::ParseVersion() {
    ParseGlobalsAddresses();

    #define TRY_ELF_TYPE(ELFT) \
    if (auto res = NPerforator::NLinguist::NPython::ParseVersion<ELFT>(File_, *GlobalsAddresses_.Get())) { \
        return res; \
    }

    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)

#undef TRY_ELF_TYPE
    return Nothing();
}

template <typename ELFT>
TMaybe<NDecode::ThreadImageOffsetType> ParseTLSPyThreadState(
    llvm::object::ObjectFile* file,
    TPythonAnalyzer::TGlobalsAddresses* addresses
) {
    llvm::object::ELFObjectFile<ELFT>* elf = llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(file);
    if (!elf) {
        return Nothing();
    }

    if (elf->getArch() != llvm::Triple::x86 && elf->getArch() != llvm::Triple::x86_64) {
        return Nothing();
    }

    if (addresses->GetCurrentThreadStateAddress == 0) {
        return Nothing();
    }

    // current_fast_get might not be inlined into GetCurrentThreadState, so we should disassemble it instead of PyThreadState_GetCurrent.
    ui64 getterAddress = (addresses->CurrentFastGetAddress != 0) ? addresses->CurrentFastGetAddress : addresses->GetCurrentThreadStateAddress;

    auto textSection = LookForSection(elf, kTextSectionName);
    if (!textSection) {
        return Nothing();
    }

    Y_LLVM_UNWRAP(sectionData, textSection->getContents(), { return Nothing(); });
    if (getterAddress < textSection->getAddress()) {
        return Nothing();
    }

    ui64 offset = getterAddress - textSection->getAddress();
    if (offset > sectionData.size()) {
        return Nothing();
    }

    TConstArrayRef<ui8> bytecode(
        reinterpret_cast<const ui8*>(sectionData.data()) + offset,
        Min(kMaxPyThreadStateGetCurrentBytecodeLength, sectionData.size() - offset)
    );

    if (addresses->CurrentFastGetAddress != 0) {
        return NDecode::NX86::DecodeCurrentFastGet(elf, bytecode);
    }

    return NDecode::NX86::DecodePyThreadStateGetCurrent(elf, bytecode);
}

TMaybe<NDecode::ThreadImageOffsetType> TPythonAnalyzer::ParseTLSPyThreadState() {
    ParseGlobalsAddresses();

    #define TRY_ELF_TYPE(ELFT) \
    if (auto res = NPerforator::NLinguist::NPython::ParseTLSPyThreadState<ELFT>(File_, GlobalsAddresses_.Get())) { \
        return res; \
    }

    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)

#undef TRY_ELF_TYPE
    return Nothing();
}

template <typename ELFT>
bool IsPythonBinary(llvm::object::ObjectFile* file) {
    llvm::object::ELFObjectFile<ELFT>* elf = llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(file);
    if (!elf) {
        return false;
    }

    for (auto&& symbol : elf->getDynamicSymbolIterators()) {
        Y_LLVM_UNWRAP(name, symbol.getName(), { continue; });

        if (TStringBuf{name.data(), name.size()} == kPyGetVersionSymbol) {
            return true;
        }
    }

    return false;
}

bool IsPythonBinary(llvm::object::ObjectFile* file) {
    #define TRY_ELF_TYPE(ELFT) \
    if (NPerforator::NLinguist::NPython::IsPythonBinary<ELFT>(file)) { \
        return true; \
    }

    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)

    #undef TRY_ELF_TYPE
    return false;
}

TMaybe<ui64> TPythonAnalyzer::ParsePyRuntimeAddress() {
    ParseGlobalsAddresses();

    if (!GlobalsAddresses_) {
        return Nothing();
    }

    if (GlobalsAddresses_->PyRuntimeAddress == 0) {
        return Nothing();
    }

    return MakeMaybe(GlobalsAddresses_->PyRuntimeAddress);
}

} // namespace NPerforator::NLinguist::NPython
