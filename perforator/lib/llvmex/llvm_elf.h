#pragma once

#include <util/generic/maybe.h>

#include <llvm/Object/ObjectFile.h>
#include <llvm/Object/ELFObjectFile.h>


#define Y_LLVM_FOR_EACH_ELF_TYPE(XX) \
    XX(llvm::object::ELF32LE) \
    XX(llvm::object::ELF32BE) \
    XX(llvm::object::ELF64LE) \
    XX(llvm::object::ELF64BE)

namespace NPerforator::NLLVM {

template <typename F>
auto VisitELF(const llvm::object::ObjectFile* file, F func) {
    using Ret = decltype(func(*llvm::dyn_cast<const llvm::object::ELFObjectFile<llvm::object::ELF64LE>>(file)));

#define TRY_ELF_TYPE(ELFT) \
    if (auto* elf = llvm::dyn_cast<const llvm::object::ELFObjectFile<ELFT>>(file)) { \
        return MakeMaybe<Ret>(func(*elf)); \
    }
    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)
#undef TRY_ELF_TYPE

    return TMaybe<Ret>{};
}

} // namespace NPerforator::NLLVM
