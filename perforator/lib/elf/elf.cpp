#include "elf.h"

#include <llvm/Object/ELFObjectFile.h>

#include <limits>

namespace NPerforator::NELF::NPrivate {

namespace {
template<typename Container>
auto MakePredicate(const Container& symbols) {
    return [&symbols](TStringBuf name) {
        for (auto s : symbols) {
            if (s == name) {
                return true;
            }
        }
        return false;
    };
}
}

TMaybe<TSymbolMap> RetrieveSymbolsFromDynsym(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        return ParseDynsym(elf, MakePredicate(symbols));
    });
}

TMaybe<TSymbolMap> RetrieveSymbolsFromSymtab(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        return ParseSymtab(elf, MakePredicate(symbols));
    });
}

TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile &file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        TSymbolMap res = ParseDynsym(elf, MakePredicate(symbols));

        llvm::SmallVector<TStringBuf> symtabSymbols;
        symtabSymbols.reserve(symbols.size());
        for (const TStringBuf& symbol : symbols) {
            if (!res.contains(symbol)) {
                symtabSymbols.push_back(symbol);
            }
        }

        TSymbolMap symtab = ParseSymtab(elf, MakePredicate(symtabSymbols));
        for (auto& [key, value] : symtab) {
            res[key] = std::move(value);
        }

        return res;
    });
}

template <typename ELFT>
bool IsElfFileImpl(const llvm::object::ObjectFile& file) {
    return llvm::dyn_cast<llvm::object::ELFObjectFile<ELFT>>(&file);
}

struct TAddressedContent {
    TConstArrayRef<ui8> Bytes;
    ui64 Flags;
};

template <typename ELFT>
TMaybe<TAddressedContent> ReadAddressedContent(
    const llvm::object::ELFObjectFile<ELFT>& file,
    const TLocation& location
) {
    const ui64 maxAddress = ELFT::Is64Bits
        ? std::numeric_limits<ui64>::max()
        : std::numeric_limits<ui32>::max();
    if (location.Size == 0 || location.Address > maxAddress || location.Size - 1 > maxAddress - location.Address) {
        return Nothing();
    }

    TMaybe<TAddressedContent> result;
    ui64 bytesBeforeNextSection = std::numeric_limits<ui64>::max();
    for (const auto& section : file.sections()) {
        const ui64 flags = llvm::object::ELFSectionRef(section).getFlags();
        if (!(flags & llvm::ELF::SHF_ALLOC) || section.isBSS()) {
            continue;
        }

        const ui64 sectionAddress = section.getAddress();
        const ui64 sectionSize = section.getSize();
        if (location.Address < sectionAddress) {
            if (sectionSize != 0) {
                bytesBeforeNextSection = Min(bytesBeforeNextSection, sectionAddress - location.Address);
            }
            continue;
        }

        const ui64 offset = location.Address - sectionAddress;
        if (offset >= sectionSize) {
            continue;
        }
        if (result) {
            return Nothing();
        }

        auto contents = section.getContents();
        if (!contents) {
            llvm::consumeError(contents.takeError());
            return Nothing();
        }
        const ui64 contentSize = contents->size();
        if (offset >= contentSize) {
            return Nothing();
        }

        const ui64 size = Min(location.Size, Min(sectionSize - offset, contentSize - offset));
        result = TAddressedContent{
            .Bytes = TConstArrayRef<ui8>(reinterpret_cast<const ui8*>(contents->data()) + offset, size),
            .Flags = flags,
        };
    }

    if (result && result->Bytes.size() > bytesBeforeNextSection) {
        return Nothing();
    }
    return result;
}

TMaybe<TAddressedContent> ReadAddressedContent(
    const llvm::object::ObjectFile& file,
    const TLocation& location
) {
    auto content = NLLVM::VisitELF(&file, [&location](const auto& elf) {
        return ReadAddressedContent(elf, location);
    });
    return content ? std::move(*content) : Nothing();
}

} // namespace NPerforator::NELF::NPrivate

namespace NPerforator::NELF {

TMaybe<TSymbolMap> RetrieveSymbols(const llvm::object::ObjectFile& file, TFunctionRef<bool(TStringBuf)> predicate) {
    using NPerforator::NELF::NPrivate::ParseDynsym;
    using NPerforator::NELF::NPrivate::ParseSymtab;

    return NLLVM::VisitELF(&file, [&predicate](const auto& elf) {
        TSymbolMap res = ParseDynsym(elf, predicate);

        TSymbolMap symtab = ParseSymtab(elf, predicate);
        for (auto& [key, value] : symtab) {
            res[key] = std::move(value);
        }

        return res;
    });
}

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

TMaybe<TConstArrayRef<ui8>> ReadBytesAtVirtualAddress(
    const llvm::object::ObjectFile& file,
    const TLocation& location
) {
    auto content = NPrivate::ReadAddressedContent(file, location);
    if (!content) {
        return Nothing();
    }
    return content->Bytes;
}

TMaybe<TConstArrayRef<ui8>> ReadCodeAtVirtualAddress(
    const llvm::object::ObjectFile& file,
    const TLocation& location
) {
    auto content = NPrivate::ReadAddressedContent(file, location);
    if (!content || !(content->Flags & llvm::ELF::SHF_EXECINSTR)) {
        return Nothing();
    }
    return content->Bytes;
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
