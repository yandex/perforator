#pragma once

#include <util/generic/string.h>
#include <util/generic/hash.h>
#include <util/generic/vector.h>
#include <util/generic/maybe.h>
#include <util/generic/array_ref.h>
#include "util/generic/function_ref.h"

#include <llvm/Object/ObjectFile.h>
#include <llvm/Object/ELFObjectFile.h>

#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>

namespace NPerforator::NELF {

struct TLocation {
    ui64 Address = 0;
    ui64 Size = 0;
};

namespace NSections {

constexpr TStringBuf kText = ".text";
constexpr TStringBuf kData = ".data";
constexpr TStringBuf kBss = ".bss";
constexpr TStringBuf kRoData = ".rodata";

} // namespace NPerforator::NELF::NSections

using TSymbolMap = THashMap<TStringBuf, TLocation>;

namespace NPrivate {

template <typename Predicate>
TSymbolMap ParseSymbolsImpl(
    const llvm::object::ELFObjectFileBase::elf_symbol_iterator_range& symbols,
    const Predicate& predicate
) {
    TSymbolMap result;

    for (const auto& symbol : symbols) {
        TLocation location;

        Y_LLVM_UNWRAP(name, symbol.getName(), { continue; });
        Y_LLVM_UNWRAP(address, symbol.getAddress(), { continue; });

        location.Address = address;
        location.Size = symbol.getSize();

        TStringBuf symbolName{name.data(), name.size()};
        if (predicate(symbolName)) {
            result[symbolName] = location;
        }
    }

    return result;
}

template <typename ELFT, typename Predicate>
TSymbolMap ParseDynsym(const llvm::object::ELFObjectFile<ELFT>& elf, const Predicate& predicate) {
    return ParseSymbolsImpl(elf.getDynamicSymbolIterators(), predicate);
}

template <typename ELFT, typename Predicate>
TSymbolMap ParseSymtab(const llvm::object::ELFObjectFile<ELFT>& elf, const Predicate& predicate) {
    return ParseSymbolsImpl(elf.symbols(), predicate);
}

TMaybe<TSymbolMap> RetrieveSymbolsFromDynsym(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

TMaybe<TSymbolMap> RetrieveSymbolsFromSymtab(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);
TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile& file, TFunctionRef<bool(TStringBuf)> predicate);

TMaybe<TSymbolMap> RetrieveSymbolsByPrefix(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbol_prefixes);

} // namespace NPerforator::NELF::NPrivate

template<typename Predicate>
TSymbolMap FilterSymbolsFromSymtab(const llvm::object::ObjectFile& file, const Predicate& predicate) {
    auto res = NLLVM::VisitELF(&file, [&predicate](const auto& elf) {
        return NPrivate::ParseSymtab(elf, predicate);
    });
    if (res) {
        return std::move(*res);
    }
    return {};
}

template <typename... Args>
TMaybe<TSymbolMap> RetrieveSymbolsFromDynsym(const llvm::object::ObjectFile& file, Args... symbols) {
    return NPerforator::NELF::NPrivate::RetrieveSymbolsFromDynsym(file, {symbols...});
}

template <typename... Args>
TMaybe<TSymbolMap> RetrieveSymbolsFromSymtab(const llvm::object::ObjectFile& file, Args... symbols) {
    return NPerforator::NELF::NPrivate::RetrieveSymbolsFromSymtab(file, {symbols...});
}

TSymbolMap RetrieveSymbolsFromSymtabChecked(
    const llvm::object::ELFObjectFile<llvm::object::ELF64LE>& elf,
    std::same_as<std::string_view> auto ...symbols
) {
    TMaybe<TSymbolMap> res = NELF::RetrieveSymbolsFromSymtab(elf, symbols...);
    Y_THROW_UNLESS(res.Defined(), "Unknown ELF kind");
    Y_THROW_UNLESS(
        res->size() == sizeof...(symbols) || res->size() == 0,
        "Found only subset of expected symbols"
    );
    return std::move(*res);
}


template <typename... Args>
TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile& file, Args... symbols) {
    return NPerforator::NELF::NPrivate::RetrieveSymbols(file, {symbols...});
}

template <typename... Args>
TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile& file, TFunctionRef<bool(TStringBuf)> predicate) {
    return NPerforator::NELF::NPrivate::RetrieveSymbols(file, predicate);
}

TMaybe<llvm::object::SectionRef> GetSection(const llvm::object::ObjectFile& file, TStringBuf sectionName);

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location,
    TStringBuf sectionName
);

bool IsElfFile(const llvm::object::ObjectFile& file);

} // namespace NPerforator::NELF
