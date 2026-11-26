#include <gtest/gtest.h>
#include <perforator/lib/python/python.h>

#include <contrib/libs/re2/re2/re2.h>
#include <contrib/libs/re2/re2/stringpiece.h>

#include <llvm/Object/ELFObjectFile.h>
#include <llvm/Support/MemoryBufferRef.h>
#include <cstring>
#include <limits>

using namespace NPerforator::NLinguist::NPython;

// Declare the TryScanVersion function from python.cpp
namespace NPerforator::NLinguist::NPython {
    TMaybe<TPythonVersion> TryScanVersion(TConstArrayRef<char> data);
} // namespace NPerforator::NLinguist::NPython

namespace {

// Copy ELF records byte-for-byte into the representation consumed by LLVM.
class TPythonELF {
public:
    using TElf = llvm::object::ELF64LE;

    TPythonELF() {
        Buffer_.resize(sizeof(TElf::Ehdr), '\0');
    }

    size_t AddSection(TStringBuf name, TStringBuf contents, ui64 address, ui64 flags, ui32 type = llvm::ELF::SHT_PROGBITS) {
        TElf::Shdr section{};
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
        return Sections_.size() - 1;
    }

    void AddSymbol(TStringBuf name, size_t section, ui64 address, ui64 size) {
        TElf::Sym symbol{};
        symbol.st_name = Strings_.size();
        Strings_.append(name.data(), name.size());
        Strings_.push_back('\0');
        symbol.st_info = llvm::ELF::STB_GLOBAL << 4;
        symbol.st_shndx = section;
        symbol.st_value = address;
        symbol.st_size = size;
        Symbols_.push_back(symbol);
    }

    std::unique_ptr<llvm::object::ObjectFile> Build() {
        const auto strings = AddSection(".strtab", Strings_, 0, 0, llvm::ELF::SHT_STRTAB);
        const auto symbols = AddSection(".symtab", {reinterpret_cast<const char*>(Symbols_.data()), Symbols_.size() * sizeof(TElf::Sym)}, 0, 0, llvm::ELF::SHT_SYMTAB);
        Sections_[symbols].sh_link = strings;
        Sections_[symbols].sh_info = 1;
        Sections_[symbols].sh_entsize = sizeof(TElf::Sym);
        const auto namesIndex = AddSection(".shstrtab", {}, 0, 0, llvm::ELF::SHT_STRTAB);
        Sections_[namesIndex].sh_size = Names_.size();
        Buffer_.append(Names_);

        Buffer_.resize((Buffer_.size() + 7) & ~size_t(7), '\0');
        TElf::Ehdr header{};
        std::memcpy(header.e_ident, "\177ELF", 4);
        header.e_ident[llvm::ELF::EI_CLASS] = llvm::ELF::ELFCLASS64;
        header.e_ident[llvm::ELF::EI_DATA] = llvm::ELF::ELFDATA2LSB;
        header.e_ident[llvm::ELF::EI_VERSION] = llvm::ELF::EV_CURRENT;
        header.e_type = llvm::ELF::ET_EXEC;
        header.e_machine = llvm::ELF::EM_X86_64;
        header.e_version = llvm::ELF::EV_CURRENT;
        header.e_ehsize = sizeof(header);
        header.e_shoff = Buffer_.size();
        header.e_shentsize = sizeof(TElf::Shdr);
        header.e_shnum = Sections_.size();
        header.e_shstrndx = namesIndex;
        Buffer_.append(reinterpret_cast<const char*>(Sections_.data()), Sections_.size() * sizeof(TElf::Shdr));
        std::memcpy(Buffer_.begin(), &header, sizeof(header));

        return Y_LLVM_RAISE(llvm::object::ObjectFile::createObjectFile(
            llvm::MemoryBufferRef(llvm::StringRef(Buffer_.data(), Buffer_.size()), "python-test")));
    }

private:
    TString Buffer_;
    TString Names_{"\0", 1};
    TString Strings_{"\0", 1};
    TVector<TElf::Shdr> Sections_{1};
    TVector<TElf::Sym> Symbols_{1};
};

} // namespace

TEST(PythonVersionParsing, VersionSymbolUsesContainingSection) {
    for (TStringBuf name : {".rodata", ".bolt.org.rodata", ".custom.python"}) {
        TPythonELF elf;
        const auto section = elf.AddSection(name, {"\xf0\x01\x0c\x03\xf0\x02\x0d\x03", 8}, 0x1000, llvm::ELF::SHF_ALLOC);
        elf.AddSymbol("Py_Version", section, 0x1004, 4);
        auto file = elf.Build();
        auto version = TPythonAnalyzer(*file).ParseVersion();
        ASSERT_TRUE(version) << name;
        EXPECT_EQ(version->Version.MajorVersion, 3);
        EXPECT_EQ(version->Version.MinorVersion, 13);
        EXPECT_EQ(version->Version.MicroVersion, 2);
        EXPECT_EQ(version->Source, EPythonVersionSource::PyVersionSymbol);
    }
}

TEST(PythonVersionParsing, RejectsInvalidVersionLocation) {
    struct TCase {
        ui64 Address;
        ui64 Size;
        ui64 Flags;
        ui32 Type;
    };
    for (const auto& test : {
             TCase{0xfff, 4, llvm::ELF::SHF_ALLOC, llvm::ELF::SHT_PROGBITS},
             TCase{0x1004, 4, llvm::ELF::SHF_ALLOC, llvm::ELF::SHT_PROGBITS},
             TCase{0x1001, 4, llvm::ELF::SHF_ALLOC, llvm::ELF::SHT_PROGBITS},
             TCase{std::numeric_limits<ui64>::max(), 4, llvm::ELF::SHF_ALLOC, llvm::ELF::SHT_PROGBITS},
             TCase{0x1000, 0, llvm::ELF::SHF_ALLOC, llvm::ELF::SHT_PROGBITS},
             TCase{0x1000, 4, 0, llvm::ELF::SHT_PROGBITS},
             TCase{0x1000, 4, llvm::ELF::SHF_ALLOC, llvm::ELF::SHT_NOBITS}}) {
        TPythonELF elf;
        const auto section = elf.AddSection(".rodata", {"\xf0\x02\x0d\x03", 4}, 0x1000, test.Flags, test.Type);
        elf.AddSymbol("Py_Version", section, test.Address, test.Size);
        auto file = elf.Build();
        EXPECT_FALSE(TPythonAnalyzer(*file).ParseVersion());
    }
}

TEST(PythonVersionParsing, RejectsOverlappingAllocatedSections) {
    TPythonELF elf;
    elf.AddSection(".overlap", {"\xf0\x0f\x07\x02", 4}, 0x1000, llvm::ELF::SHF_ALLOC);
    const auto section = elf.AddSection(".rodata", {"\xf0\x02\x0d\x03", 4}, 0x1000, llvm::ELF::SHF_ALLOC);
    elf.AddSymbol("Py_Version", section, 0x1000, 4);
    auto file = elf.Build();
    EXPECT_FALSE(TPythonAnalyzer(*file).ParseVersion());
}

TEST(PythonVersionParsing, RejectsPartiallyOverlappingAllocatedSections) {
    for (ui64 start : {0xffe, 0xfff, 0x1001, 0x1002, 0x1003}) {
        TPythonELF elf;
        elf.AddSection(".overlap", {"\x11\x22\x33\x44", 4}, start, llvm::ELF::SHF_ALLOC);
        const auto section = elf.AddSection(".rodata", {"\xf0\x02\x0d\x03", 4}, 0x1000, llvm::ELF::SHF_ALLOC);
        elf.AddSymbol("Py_Version", section, 0x1000, 4);
        auto file = elf.Build();
        EXPECT_FALSE(TPythonAnalyzer(*file).ParseVersion()) << start;
    }
}

TEST(PythonVersionParsing, AllowsAdjacentAndEmptyAllocatedSections) {
    for (ui64 start : {0xffc, 0x1004}) {
        TPythonELF elf;
        elf.AddSection(".neighbor", {"\x11\x22\x33\x44", 4}, start, llvm::ELF::SHF_ALLOC);
        elf.AddSection(".empty", {}, 0x1002, llvm::ELF::SHF_ALLOC);
        const auto section = elf.AddSection(".rodata", {"\xf0\x02\x0d\x03", 4}, 0x1000, llvm::ELF::SHF_ALLOC);
        elf.AddSymbol("Py_Version", section, 0x1000, 4);
        auto file = elf.Build();
        const auto version = TPythonAnalyzer(*file).ParseVersion();
        ASSERT_TRUE(version);
        EXPECT_EQ(version->Version.MajorVersion, 3);
        EXPECT_EQ(version->Version.MinorVersion, 13);
        EXPECT_EQ(version->Version.MicroVersion, 2);
    }
}

#if defined(__x86_64__)
TEST(PythonVersionParsing, GetVersionUsesRenamedCodeAndDataSections) {
    for (ui64 codeFlags : {ui64(llvm::ELF::SHF_ALLOC), ui64(llvm::ELF::SHF_ALLOC | llvm::ELF::SHF_EXECINSTR)}) {
        TPythonELF elf;
        elf.AddSection(".debug_overlap", "2.7.15", 0x1000, 0);
        elf.AddSection(".bolt.org.rodata", "3.13.2", 0x1000, llvm::ELF::SHF_ALLOC);
        // mov $0x1000, %ecx; ret
        const auto code = elf.AddSection(".bolt.text", {"\xb9\x00\x10\x00\x00\xc3", 6}, 0x2000, codeFlags);
        elf.AddSymbol("Py_GetVersion", code, 0x2000, 0);
        auto file = elf.Build();
        auto version = TPythonAnalyzer(*file).ParseVersion();
        if (!(codeFlags & llvm::ELF::SHF_EXECINSTR)) {
            EXPECT_FALSE(version);
            continue;
        }
        ASSERT_TRUE(version);
        EXPECT_EQ(version->Version.MajorVersion, 3);
        EXPECT_EQ(version->Version.MinorVersion, 13);
        EXPECT_EQ(version->Version.MicroVersion, 2);
        EXPECT_EQ(version->Source, EPythonVersionSource::PyGetVersionDisassembly);
    }
}

TEST(PythonVersionParsing, ThreadStateUsesRenamedCodeSection) {
    for (ui64 codeFlags : {ui64(llvm::ELF::SHF_ALLOC), ui64(llvm::ELF::SHF_ALLOC | llvm::ELF::SHF_EXECINSTR)}) {
        TPythonELF elf;
        // mov %fs:-8, %rax; ret
        const auto code = elf.AddSection(".custom.text", {"\x64\x48\x8b\x04\x25\xf8\xff\xff\xff\xc3", 10}, 0x2000, codeFlags);
        elf.AddSymbol("_PyThreadState_GetCurrent", code, 0x2000, 0);
        auto file = elf.Build();
        auto offset = TPythonAnalyzer(*file).ParseTLSPyThreadState();
        if (!(codeFlags & llvm::ELF::SHF_EXECINSTR)) {
            EXPECT_FALSE(offset);
            continue;
        }
        ASSERT_TRUE(offset);
        EXPECT_EQ(*offset, -8);
    }
}

TEST(PythonVersionParsing, AutoTSSKeyUsesRenamedCodeSection) {
    for (ui64 codeFlags : {ui64(llvm::ELF::SHF_ALLOC), ui64(llvm::ELF::SHF_ALLOC | llvm::ELF::SHF_EXECINSTR)}) {
        TPythonELF elf;
        // mov $0x1234, %edi; call next instruction
        const auto code = elf.AddSection(".custom.text", {"\xbf\x34\x12\x00\x00\xe8\x00\x00\x00\x00", 10}, 0x2000, codeFlags);
        elf.AddSymbol("PyGILState_Ensure", code, 0x2000, 0);
        auto file = elf.Build();
        auto address = TPythonAnalyzer(*file).ParseAutoTSSKeyAddress();
        if (!(codeFlags & llvm::ELF::SHF_EXECINSTR)) {
            EXPECT_FALSE(address);
            continue;
        }
        ASSERT_TRUE(address);
        EXPECT_EQ(*address, 0x1234);
    }
}

TEST(PythonVersionParsing, InterpreterHeadUsesRenamedCodeSection) {
    for (ui64 codeFlags : {ui64(llvm::ELF::SHF_ALLOC), ui64(llvm::ELF::SHF_ALLOC | llvm::ELF::SHF_EXECINSTR)}) {
        TPythonELF elf;
        // mov $0x12345678, %eax; ret
        const auto code = elf.AddSection(".custom.text", {"\xb8\x78\x56\x34\x12\xc3", 6}, 0x2000, codeFlags);
        elf.AddSymbol("PyInterpreterState_Head", code, 0x2000, 0);
        auto file = elf.Build();
        auto address = TPythonAnalyzer(*file).ParseInterpHeadAddress();
        if (!(codeFlags & llvm::ELF::SHF_EXECINSTR)) {
            EXPECT_FALSE(address);
            continue;
        }
        ASSERT_TRUE(address);
        EXPECT_EQ(*address, 0x12345678);
    }
}
#endif

TEST(PythonVersionParsing, TryScanVersion) {
    struct TTestCase {
        TString VersionString;
        bool ShouldParse;
        TPythonVersion ExpectedVersion;
    };

    TVector<TTestCase> testCases = {
        {"Python 3.10.0", true, {3, 10, 0}},
        {"Python 3.10.1", true, {3, 10, 1}},

        {"Python 2.7", true, {2, 7, 0}},
        {"Python 2.7.10", true, {2, 7, 10}},
        {"Python 3.8.15", true, {3, 8, 15}},
        {"Python 3.12.0", true, {3, 12, 0}},
        {"Python 3.12.1", true, {3, 12, 1}},
        {"Python 2.8.3", true, {2, 8, 3}},

        {"Python version 3.10.0, 64 bit", true, {3, 10, 0}},
        {"Version: Python 2.7.15", true, {2, 7, 15}},

        {"Python 4.0.0", false, {}}, // Major not 2 or 3
        {"No version here", false, {}},
        {"Python v9.8.7", false, {}},
    };

    for (const auto& testCase : testCases) {
        TMaybe<TPythonVersion> result = TryScanVersion(TConstArrayRef<char>{testCase.VersionString.data(), testCase.VersionString.size()});

        if (testCase.ShouldParse) {
            ASSERT_TRUE(result.Defined()) << "Failed to parse version from string: '" << testCase.VersionString << "'";

            TString expectedVersionStr =
                std::to_string(testCase.ExpectedVersion.MajorVersion) + "." +
                std::to_string(testCase.ExpectedVersion.MinorVersion) + "." +
                std::to_string(testCase.ExpectedVersion.MicroVersion);

            TString actualVersionStr =
                std::to_string(result->MajorVersion) + "." +
                std::to_string(result->MinorVersion) + "." +
                std::to_string(result->MicroVersion);

            EXPECT_EQ(result->MajorVersion, testCase.ExpectedVersion.MajorVersion)
                << "Major version mismatch for input: '" << testCase.VersionString
                << "', expected: " << expectedVersionStr
                << ", got: " << actualVersionStr;

            EXPECT_EQ(result->MinorVersion, testCase.ExpectedVersion.MinorVersion)
                << "Minor version mismatch for input: '" << testCase.VersionString
                << "', expected: " << expectedVersionStr
                << ", got: " << actualVersionStr;

            EXPECT_EQ(result->MicroVersion, testCase.ExpectedVersion.MicroVersion)
                << "Micro version mismatch for input: '" << testCase.VersionString
                << "', expected: " << expectedVersionStr
                << ", got: " << actualVersionStr;
        } else {
            EXPECT_FALSE(result.Defined()) << "Should not parse version from string: '" << testCase.VersionString << "'";
        }
    }
}
