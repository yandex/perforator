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
constexpr static std::string_view kVersion = "_ZN19Abstract_VM_Version17_vm_major_versionE";

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
        kVersion
    );
    if (symbols.empty()) {
        return std::nullopt;
    }
    analysis.Cheatsheet.set_code_cache_heaps(symbols.at(kCodeCacheHeapsSym).Address);
    analysis.Cheatsheet.set_abstract_interpreter_code(symbols.at(kAbstractInterpreterCodeSym).Address);

    auto data = NELF::RetrieveContentFromSection(elf, symbols.at(kVersion), NELF::NSections::kData);
    if (!data) {
        ythrow yexception() << "failed to get version info";
    }
    constexpr int kIntSize = sizeof(ui32);
    if (data->size() != kIntSize) {
        ythrow yexception() << "unexpected version size " << data->size();
    }
    std::memcpy(&analysis.Version, data->data(), kIntSize);

    return analysis;
}


} // namespace NPerforator::NLinguist::NJvm
