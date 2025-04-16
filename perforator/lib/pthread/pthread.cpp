#include "pthread.h"

#include <library/cpp/logger/global/global.h>

#include <perforator/lib/elf/elf.h>
#include <perforator/lib/pthread/asm/x86/decode.h>

#include <perforator/lib/llvmex/llvm_exception.h>

namespace NPerforator::NPthread {

TLibPthreadAnalyzer::TLibPthreadAnalyzer(const llvm::object::ObjectFile& file) : File_(file) {}

void TLibPthreadAnalyzer::ParseSymbolLocations() {
    if (Symbols_) {
        return;
    }

    Symbols_ = MakeHolder<TSymbols>();
    auto dynamicSymbols = NELF::RetrieveDynamicSymbols(File_, kPthreadGetspecificSymbol);
    if (dynamicSymbols) {
        Symbols_->PthreadGetspecific = (*dynamicSymbols)[kPthreadGetspecificSymbol];
    }
}

TMaybe<TAccessTSSInfo> ParseAccessTSSInfoImpl(
    const llvm::object::ObjectFile& elf,
    const NPerforator::NELF::TLocation& pthreadGetspecific
) {
    if (pthreadGetspecific.Address == 0) {
        return Nothing();
    }

    NPerforator::NELF::TLocation symbolLocation{.Address = pthreadGetspecific.Address, .Size = 200};
    if (pthreadGetspecific.Size > 0) {
        symbolLocation.Size = pthreadGetspecific.Size;
    }

    auto bytecode = NPerforator::NELF::RetrieveContentFromTextSection(elf, symbolLocation);
    if (!bytecode) {
        return Nothing();
    }

    auto expectedResult = NAsm::NX86::DecodePthreadGetspecific(elf.makeTriple(), *bytecode);

    if (!expectedResult) {
        return Nothing();
    }

    return MakeMaybe(std::move(expectedResult.value()));
}

TMaybe<TAccessTSSInfo> TLibPthreadAnalyzer::ParseAccessTSSInfo() {
    ParseSymbolLocations();

    if (!Symbols_) {
        return Nothing();
    }

    if (!Symbols_->PthreadGetspecific) {
        return Nothing();
    }

    return ParseAccessTSSInfoImpl(File_, *Symbols_->PthreadGetspecific);
}

bool IsLibPthreadBinary(const llvm::object::ObjectFile& file) {
    auto dynamicSymbols = NELF::RetrieveDynamicSymbols(file, kPthreadGetspecificSymbol, kPthreadSetspecificSymbol);
    if (!dynamicSymbols) {
        return false;
    }

    // Symbols can be found in .dynsym when they are linked from dynamic libraries, so check for address
    return (dynamicSymbols->size() == 2 &&
        (*dynamicSymbols)[kPthreadGetspecificSymbol].Address != 0 &&
        (*dynamicSymbols)[kPthreadSetspecificSymbol].Address != 0);
}

} // namespace NPerforator::NPthread
