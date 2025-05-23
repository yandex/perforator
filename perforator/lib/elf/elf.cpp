#include "elf.h"

#include <perforator/lib/llvmex/llvm_elf.h>
#include <perforator/lib/llvmex/llvm_exception.h>

#include <llvm/Object/ELFObjectFile.h>

namespace NPerforator::NELF::NPrivate {

THashMap<TStringBuf, TLocation> ParseSymbolsImpl(
    const llvm::object::ELFObjectFileBase::elf_symbol_iterator_range& symbols,
    std::initializer_list<TStringBuf> targetSymbols
) {
    THashMap<TStringBuf, TLocation> result;

    for (const auto& symbol : symbols) {
        TLocation location;

        Y_LLVM_UNWRAP(name, symbol.getName(), { continue; });
        Y_LLVM_UNWRAP(address, symbol.getAddress(), { continue; });

        location.Address = address;
        location.Size = symbol.getSize();

        TStringBuf symbolName{name.data(), name.size()};
        for (const auto& targetSymbol : targetSymbols) {
            if (symbolName == targetSymbol) {
                result[symbolName] = location;
                break;
            }
        }
    }

    return result;
}

template <typename ELFT>
THashMap<TStringBuf, TLocation> ParseDynamicSymbols(const llvm::object::ELFObjectFile<ELFT>& elf, std::initializer_list<TStringBuf> symbols) {
    return ParseSymbolsImpl(elf.getDynamicSymbolIterators(), symbols);
}

template <typename ELFT>
THashMap<TStringBuf, TLocation> ParseSymbols(const llvm::object::ELFObjectFile<ELFT>& elf, std::initializer_list<TStringBuf> symbols) {
    return ParseSymbolsImpl(elf.symbols(), symbols);
}

TMaybe<THashMap<TStringBuf, TLocation>> RetrieveDynamicSymbols(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        return ParseDynamicSymbols(elf, symbols);
    });
}

TMaybe<THashMap<TStringBuf, TLocation>> RetrieveSymbols(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        return ParseSymbols(elf, symbols);
    });
}

template <typename ELFT>
bool IsElfFileImpl(const llvm::object::ObjectFile& file) {
    return llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(&file);
}

} // namespace NPerforator::NELF::NPrivate

namespace NPerforator::NELF {

TMaybe<llvm::object::SectionRef> GetSection(const llvm::object::ObjectFile& file, TStringBuf sectionName) {
    for (const auto& section : file.sections()) {
        Y_LLVM_UNWRAP(name, section.getName(), { continue; });
        if (TStringBuf{name.data(), name.size()} == sectionName) {
            return section;
        }
    }

    return Nothing();
}

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location,
    TStringBuf sectionName
) {
    auto section = GetSection(file, sectionName);
    if (!section) {
        return Nothing();
    }

    Y_LLVM_UNWRAP(sectionData, section->getContents(), { return Nothing(); });

    if (location.Address < section->getAddress()) {
        return Nothing();
    }

    ui64 offset = location.Address - section->getAddress();
    if (offset >= sectionData.size()) {
        return Nothing();
    }

    size_t contentSize = Min<size_t>(location.Size, sectionData.size() - offset);

    return MakeMaybe(TConstArrayRef<ui8>(
        static_cast<const ui8*>(reinterpret_cast<const unsigned char*>(sectionData.data()) + offset),
        contentSize
    ));
}

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromTextSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location
) {
    return RetrieveContentFromSection(file, location, NSections::kTextSectionName);
}

TMaybe<TConstArrayRef<ui8>> RetrieveContentFromRodataSection(
    const llvm::object::ObjectFile& file,
    const TLocation& location
) {
    return RetrieveContentFromSection(file, location, NSections::kRoDataSectionName);
}

bool IsElfFile(const llvm::object::ObjectFile &file) {
#define TRY_ELF_TYPE(ELFT)                 \
if (NPrivate::IsElfFileImpl<ELFT>(file)) { \
    return true;                           \
}

    Y_LLVM_FOR_EACH_ELF_TYPE(TRY_ELF_TYPE)

#undef TRY_ELF_TYPE
    return false;
}

} // namespace NPerforator::NELF
