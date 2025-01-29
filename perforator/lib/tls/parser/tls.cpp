#include "tls.h"

#include <perforator/lib/tls/variable.h>
#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <util/generic/array_ref.h>
#include <util/generic/maybe.h>
#include <util/stream/format.h>

#include <llvm/Demangle/Demangle.h>
#include <llvm/Object/ELF.h>
#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Object/ObjectFile.h>


namespace NPerforator::NThreadLocal {

template <typename ELFT>
bool VisitTlsVariables(
    llvm::object::ObjectFile* file,
    TFunctionRef<void(const TTlsParser::TVariableRef&)> callback
) {
    using Elf_Phdr_Range = typename ELFT::PhdrRange;

    llvm::object::ELFObjectFile<ELFT>* elf = llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(file);
    if (!elf) {
        return false;
    }

    llvm::Expected<Elf_Phdr_Range> range = elf->getELFFile().program_headers();
    if (!range) {
        return false;
    }

    i64 imageSize = 0;
    for (auto&& phdr : *range) {
        if (phdr.p_type != llvm::ELF::PT_TLS) {
            continue;
        }

        ui64 memsize = phdr.p_memsz;
        ui64 align = phdr.p_align;
        if (!IsPowerOf2(align)) {
            continue;
        }
        imageSize = AlignUp(memsize, align);
    }

    for (auto&& symbol : elf->symbols()) {
        if (symbol.getELFType() != llvm::ELF::STT_TLS) {
            continue;
        }

        Y_LLVM_UNWRAP(name, symbol.getName(), { continue; });
        Y_LLVM_UNWRAP(address, symbol.getAddress(), { continue; });

        auto demangled = llvm::demangle({name.data(), name.size()});
        if (!demangled.contains(Y_PERFORATOR_VARIABLE_NAME_PREFIX_STRING)) {
            continue;
        }

        callback(TTlsParser::TVariableRef{
            .ThreadImageOffset = imageSize - static_cast<i64>(address),
            .Name = demangled,
        });
    }

    return true;
}

TTlsParser::TTlsParser(llvm::object::ObjectFile* file)
    : File_{file}
{
}

void TTlsParser::VisitVariables(TFunctionRef<void(const TVariableRef&)> callback) {
#define TRY_ELF_TYPE(ELFT) \
    if (auto res = VisitTlsVariables<ELFT>(File_, callback)) { \
        return; \
    }

    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)

#undef TRY_ELF_TYPE
    return;
}

} // namespace NPerforator::NThreadLocal
