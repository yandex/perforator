#include "elf.h"


#include <llvm/Object/ELFObjectFile.h>

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

TMaybe<TSymbolMap> RetrieveSymbolsFromDynsymByPrefix(const llvm::object::ObjectFile& file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        return ParseDynsymByPrefix(elf, symbols);
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

TMaybe<TSymbolMap> RetrieveSymbolsByPrefix(const llvm::object::ObjectFile &file, std::initializer_list<TStringBuf> symbols) {
    return NLLVM::VisitELF(&file, [&symbols](const auto& elf) {
        TSymbolMap res = ParseDynsymByPrefix(elf, symbols);

        llvm::SmallVector<TStringBuf> symtabSymbols;
        symtabSymbols.reserve(symbols.size());
        for (const TStringBuf& symbol : symbols) {
            bool found = false;
            for (auto& [key, value] : res) {
                if (key.find(symbol) == 0) {
                    found = true;
                }
            }
            if (!found) {
                symtabSymbols.push_back(symbol);
            }
        }

        TSymbolMap symtab = ParseSymtabByPrefix(elf, symtabSymbols);
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
