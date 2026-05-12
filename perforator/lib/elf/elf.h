#pragma once

#include <util/generic/string.h>
#include <util/generic/hash.h>
#include <util/generic/vector.h>
#include <util/generic/maybe.h>
#include <util/generic/array_ref.h>

#include <llvm/Object/ObjectFile.h>
#include <llvm/Object/ELFObjectFile.h>

#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <cstdlib>

namespace NPerforator::NELF {

struct TLocation {
    ui64 Address = 0;
    ui64 Size = 0;
};

namespace NSections {

constexpr TStringBuf kTextSectionName = ".text";
constexpr TStringBuf kDataSectionName = ".data";
constexpr TStringBuf kBssSectionName = ".bss";
constexpr TStringBuf kRoDataSectionName = ".rodata";

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

template <typename Container>
TSymbolMap ParseSymbolsByPrefixImpl(
    const llvm::object::ELFObjectFileBase::elf_symbol_iterator_range& symbols,
    const Container& targetSymbols
) {
    fprintf(stderr, "SPAR: %s\n", __PRETTY_FUNCTION__); fflush(stderr);

    TSymbolMap result;

    for (const auto& symbol : symbols) {
        TLocation location;

        Y_LLVM_UNWRAP(name, symbol.getName(), { continue; });
        Y_LLVM_UNWRAP(address, symbol.getAddress(), { continue; });

        location.Address = address;
        location.Size = symbol.getSize();

        TStringBuf symbolName{name.data(), name.size()};
        for (const auto& targetSymbol : targetSymbols) {
            if (symbolName.StartsWith(targetSymbol)) {
                result[symbolName] = location;
                break;
            }
        }
    }

    return result;
}

template <typename ELFT, typename Predicate>
TSymbolMap ParseDynsym(const llvm::object::ELFObjectFile<ELFT>& elf, const Predicate& predicate) {
    return ParseSymbolsImpl(elf.getDynamicSymbolIterators(), predicate);
}

template <typename ELFT, typename Container>
TSymbolMap ParseDynsymByPrefix(const llvm::object::ELFObjectFile<ELFT>& elf, const Container& symbols) {
    fprintf(stderr, "SPAR: %s\n", __PRETTY_FUNCTION__); fflush(stderr);

    return ParseSymbolsByPrefixImpl(elf.getDynamicSymbolIterators(), symbols);
}

template <typename ELFT, typename Predicate>
TSymbolMap ParseSymtab(const llvm::object::ELFObjectFile<ELFT>& elf, const Predicate& predicate) {
    return ParseSymbolsImpl(elf.symbols(), predicate);
}

template <typename ELFT, typename Container>
TSymbolMap ParseSymtabByPrefix(const llvm::object::ELFObjectFile<ELFT>& elf, const Container& symbols) {
    fprintf(stderr, "SPAR: %s\n", __PRETTY_FUNCTION__); fflush(stderr);

    return ParseSymbolsByPrefixImpl(elf.symbols(), symbols);
}

TMaybe<TSymbolMap> RetrieveSymbolsFromDynsym(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

TMaybe<TSymbolMap> RetrieveSymbolsFromDynsymByPrefix(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

TMaybe<TSymbolMap> RetrieveSymbolsFromSymtab(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

TMaybe<TSymbolMap> RetrieveSymbolsByPrefix(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols);

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
TMaybe<TSymbolMap> RetrieveSymbolsFromDynsymByPrefix(const llvm::object::ObjectFile& file, Args... symbols) {
    fprintf(stderr, "SPAR: %s\n", __PRETTY_FUNCTION__); fflush(stderr);

    return NPerforator::NELF::NPrivate::RetrieveSymbolsFromDynsymByPrefix(file, {symbols...});
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
TMaybe<TSymbolMap> RetrieveSymbolsByPrefix(const llvm::object::ObjectFile& file, Args... symbols) {
    fprintf(stderr, "SPAR: %s\n", __PRETTY_FUNCTION__); fflush(stderr);

    return NPerforator::NELF::NPrivate::RetrieveSymbolsByPrefix(file, {symbols...});
}

TMaybe<llvm::object::SectionRef> GetSection(const llvm::object::ObjectFile& file, TStringBuf sectionName);

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location,
    TStringBuf sectionName
);

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromTextSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location
);

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromRodataSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location
);

bool IsElfFile(const llvm::object::ObjectFile& file);

} // namespace NPerforator::NELF
