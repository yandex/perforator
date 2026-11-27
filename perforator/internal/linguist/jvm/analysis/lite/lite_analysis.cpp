#include "lite_analysis.h"

#include <perforator/lib/elf/elf.h>

#include <llvm/Object/ObjectFile.h>
#include <llvm/Object/ELF.h>
#include <llvm/Object/ELFObjectFile.h>

#include <util/generic/yexception.h>
#include <util/stream/output.h>


namespace NPerforator::NLinguist::NJvm {

namespace {

// used during minimal analysis
constexpr static std::string_view kAbstractInterpreterCodeSym = "_ZN19AbstractInterpreter5_codeE";
constexpr static std::string_view kCodeCacheHeapsSym = "_ZN9CodeCache6_heapsE";
constexpr static std::string_view kMajorVersion = "_ZN19Abstract_VM_Version17_vm_major_versionE";
constexpr static std::string_view kMinorVersion = "_ZN19Abstract_VM_Version17_vm_minor_versionE";
constexpr static std::string_view kSecurityVersion = "_ZN19Abstract_VM_Version20_vm_security_versionE";

}

std::optional<TJvmAnalysis> ProcessJvmBinaryMinimal(const llvm::object::ObjectFile& binary) {
    TJvmAnalysis analysis;
    const llvm::object::ELFObjectFile<llvm::object::ELF64LE>* elfPtr = llvm::dyn_cast<llvm::object::ELFObjectFile<llvm::object::ELF64LE>>(&binary);
    Y_THROW_UNLESS(elfPtr != nullptr);
    const llvm::object::ELFObjectFile<llvm::object::ELF64LE>& elf = *elfPtr;
    NELF::TSymbolMap symbols = NELF::RetrieveSymbolsFromSymtabChecked(
        elf,
        kCodeCacheHeapsSym,
        kAbstractInterpreterCodeSym,
        kMajorVersion,
        kMinorVersion,
        kSecurityVersion
    );
    if (symbols.empty()) {
        return std::nullopt;
    }
    analysis.Cheatsheet.set_code_cache_heaps(symbols.at(kCodeCacheHeapsSym).Address);
    analysis.Cheatsheet.set_abstract_interpreter_code(symbols.at(kAbstractInterpreterCodeSym).Address);

    auto parseInt = [&elf, &symbols](std::string_view sym) -> ui32 {
        auto data = NELF::RetrieveContentFromSection(elf, symbols.at(sym), NELF::NSections::kData);
        if (!data) {
            return 0;
        }
        constexpr int kIntSize = sizeof(ui32);
        if (data->size() != kIntSize) {
            ythrow yexception() << "unexpected version size " << data->size();
        }
        ui32 res;
        std::memcpy(&res, data->data(), kIntSize);
        return res;
    };
    analysis.Version.set_major(parseInt(kMajorVersion));
    analysis.Version.set_minor(parseInt(kMinorVersion));
    analysis.Version.set_security(parseInt(kSecurityVersion));

    return analysis;
}


} // namespace NPerforator::NLinguist::NJvm
