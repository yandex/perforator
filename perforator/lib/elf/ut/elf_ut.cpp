#include <perforator/lib/elf/elf.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Support/MemoryBufferRef.h>

#include <cstring>
#include <limits>

namespace NPerforator::NELF {
namespace {

template <typename TElf>
class TTestElf {
public:
    TTestElf() {
        Buffer_.resize(sizeof(typename TElf::Ehdr), '\0');
    }

    void AddSection(TStringBuf name, TStringBuf contents, ui64 address, ui64 flags, ui32 type = llvm::ELF::SHT_PROGBITS) {
        typename TElf::Shdr section{};
        section.sh_name = Names_.size();
        Names_.append(name.data(), name.size());
        Names_.push_back('\0');
        section.sh_addr = address;
        section.sh_flags = flags;
        section.sh_type = type;
        section.sh_offset = Buffer_.size();
        section.sh_size = contents.size();
        section.sh_addralign = 1;
        Buffer_.append(contents.data(), contents.size());
        Sections_.push_back(section);
    }

    std::unique_ptr<llvm::object::ObjectFile> Build() {
        const size_t namesIndex = Sections_.size();
        AddSection(".shstrtab", {}, 0, 0, llvm::ELF::SHT_STRTAB);
        Sections_[namesIndex].sh_size = Names_.size();
        Buffer_.append(Names_);

        Buffer_.resize((Buffer_.size() + 7) & ~size_t(7), '\0');
        typename TElf::Ehdr header{};
        std::memcpy(header.e_ident, "\177ELF", 4);
        header.e_ident[llvm::ELF::EI_CLASS] = TElf::Is64Bits ? llvm::ELF::ELFCLASS64 : llvm::ELF::ELFCLASS32;
        header.e_ident[llvm::ELF::EI_DATA] = llvm::ELF::ELFDATA2LSB;
        header.e_ident[llvm::ELF::EI_VERSION] = llvm::ELF::EV_CURRENT;
        header.e_type = llvm::ELF::ET_EXEC;
        header.e_machine = TElf::Is64Bits ? llvm::ELF::EM_X86_64 : llvm::ELF::EM_386;
        header.e_version = llvm::ELF::EV_CURRENT;
        header.e_ehsize = sizeof(header);
        header.e_shoff = Buffer_.size();
        header.e_shentsize = sizeof(typename TElf::Shdr);
        header.e_shnum = Sections_.size();
        header.e_shstrndx = namesIndex;
        Buffer_.append(reinterpret_cast<const char*>(Sections_.data()), Sections_.size() * sizeof(typename TElf::Shdr));
        std::memcpy(Buffer_.begin(), &header, sizeof(header));

        return Y_LLVM_RAISE(llvm::object::ObjectFile::createObjectFile(
            llvm::MemoryBufferRef(llvm::StringRef(Buffer_.data(), Buffer_.size()), "elf-test")));
    }

private:
    TString Buffer_;
    TString Names_{"\0", 1};
    TVector<typename TElf::Shdr> Sections_{1};
};

template <typename TElf>
void TestAddressLimit() {
    const ui64 maxAddress = TElf::Is64Bits
        ? std::numeric_limits<ui64>::max()
        : std::numeric_limits<ui32>::max();

    TTestElf<TElf> elf;
    elf.AddSection(".data", "abcd", maxAddress - 3, llvm::ELF::SHF_ALLOC);
    auto file = elf.Build();

    const auto lastByte = ReadBytesAtVirtualAddress(*file, {.Address = maxAddress, .Size = 1});
    ASSERT_TRUE(lastByte);
    EXPECT_EQ(lastByte->size(), 1);
    EXPECT_FALSE(ReadBytesAtVirtualAddress(*file, {.Address = maxAddress, .Size = 2}));
    const auto lastWord = ReadBytesAtVirtualAddress(*file, {.Address = maxAddress - 3, .Size = 4});
    ASSERT_TRUE(lastWord);
    EXPECT_EQ(lastWord->size(), 4);
    EXPECT_FALSE(ReadBytesAtVirtualAddress(*file, {.Address = maxAddress - 2, .Size = 4}));
    EXPECT_FALSE(ReadBytesAtVirtualAddress(*file, {.Address = maxAddress, .Size = 0}));
}

TEST(ElfVirtualAddressRead, EnforcesElfAddressWidth) {
    TestAddressLimit<llvm::object::ELF32LE>();
    TestAddressLimit<llvm::object::ELF64LE>();

    TTestElf<llvm::object::ELF32LE> elf;
    elf.AddSection(".data", "abcd", 0x1000, llvm::ELF::SHF_ALLOC);
    auto file = elf.Build();
    EXPECT_FALSE(ReadBytesAtVirtualAddress(*file, {.Address = ui64{1} << 32, .Size = 1}));
}

TEST(ElfVirtualAddressRead, RequiresExecutableSectionForCode) {
    TTestElf<llvm::object::ELF64LE> elf;
    elf.AddSection(".data", "abcd", 0x1000, llvm::ELF::SHF_ALLOC);
    auto file = elf.Build();

    ASSERT_TRUE(ReadBytesAtVirtualAddress(*file, {.Address = 0x1000, .Size = 4}));
    EXPECT_FALSE(ReadCodeAtVirtualAddress(*file, {.Address = 0x1000, .Size = 4}));
}

TEST(ElfVirtualAddressRead, RejectsAmbiguousMappingsBeforeCheckingExecutableFlag) {
    for (ui64 dataAddress : {0x1000, 0x1002}) {
        for (bool executableFirst : {false, true}) {
            TTestElf<llvm::object::ELF64LE> elf;
            if (executableFirst) {
                elf.AddSection(".text", "abcd", 0x1000, llvm::ELF::SHF_ALLOC | llvm::ELF::SHF_EXECINSTR);
                elf.AddSection(".data", "abcd", dataAddress, llvm::ELF::SHF_ALLOC);
            } else {
                elf.AddSection(".data", "abcd", dataAddress, llvm::ELF::SHF_ALLOC);
                elf.AddSection(".text", "abcd", 0x1000, llvm::ELF::SHF_ALLOC | llvm::ELF::SHF_EXECINSTR);
            }
            auto file = elf.Build();
            EXPECT_FALSE(ReadBytesAtVirtualAddress(*file, {.Address = 0x1000, .Size = 4}));
            EXPECT_FALSE(ReadCodeAtVirtualAddress(*file, {.Address = 0x1000, .Size = 4}));
        }
    }
}

} // namespace
} // namespace NPerforator::NELF
