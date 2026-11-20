#include <perforator/agent/collector/nativepipeline/raw_sample.h>

#include <library/cpp/testing/unittest/registar.h>

#include <util/string/hex.h>

#include <cstdint>
#include <cstring>

namespace NPerforator::NAgent::NNativePipeline {
namespace {

////////////////////////////////////////////////////////////////////////////////

constexpr size_t PythonFrameSize = 32;
constexpr size_t PythonSectionSize = sizeof(language_section_header) + PythonFrameSize;

TString MakeRecord() {
    record_sample_header header{};
    header.tag = RECORD_TAG_SAMPLE;
    header.sample_type = SAMPLE_TYPE_PERF_EVENT;
    header.kern_stack = {.offset = 0, .size = 2 * sizeof(std::uint64_t)};
    header.user_stack = {.offset = 16, .size = sizeof(std::uint64_t)};
    header.lbr = {.offset = 24, .size = 0};
    header.tls = {.offset = 24, .size = 0};
    header.cgroups = {.offset = 24, .size = sizeof(std::uint64_t)};
    header.language_sections = {.offset = 32, .size = PythonSectionSize};

    TString record(sizeof(header) + 32 + PythonSectionSize, '\0');
    std::memcpy(record.begin(), &header, sizeof(header));

    const std::uint64_t kernel[] = {0x10, 0x20};
    const std::uint64_t user = 0x30;
    const std::uint64_t cgroup = 42;
    std::memcpy(record.begin() + sizeof(header), kernel, sizeof(kernel));
    std::memcpy(record.begin() + sizeof(header) + 16, &user, sizeof(user));
    std::memcpy(record.begin() + sizeof(header) + 24, &cgroup, sizeof(cgroup));

    language_section_header language{
        .byte_size = PythonFrameSize,
        .language = LANGUAGE_PYTHON,
        .element_size = PythonFrameSize,
    };
    struct {
        std::uint64_t ObjectAddress;
        std::int32_t LineStart;
        std::uint32_t Padding;
    } frame{
        .ObjectAddress = 0x1234,
        .LineStart = 11,
        .Padding = 0,
    };
    std::memcpy(record.begin() + sizeof(header) + 32, &language, sizeof(language));
    std::memcpy(record.begin() + sizeof(header) + 40, &frame, sizeof(frame));
    return record;
}

Y_UNIT_TEST_SUITE(RawSample) {
    Y_UNIT_TEST(WireFormat) {
        // Same bytes and mutations as TestWireFormat in wire_fixtures_test.go.
        // Header, kernel/user stacks, LBR, TLS, cgroups, then Python/PHP/JVM/Lua.
        const TStringBuf hex =
            "0000000001000000000000000000000000000000000000000000000000000000"
            "01000300e8030000393000000000000000000000000000000000000000000000"
            "000000000000000000000000000000002a0000002b0000000000000000000000"
            "00000000000000002b0200000000000064000000000000000100000000000000"
            "0100000000000000c80000000000000000001000100008001800300048003001"
            "780108008001a000100000000000000020000000000000003000000000000000"
            "4000000000000000500000000000000007000000000000006000000000000000"
            "7000000000000000090000000000000040000000000000000100000000000000"
            "6300000000000000000000000000000000000000000000000000000000000000"
            "0000000000000000000000000000000000000000000000000000000000000000"
            "0000000000000000000000000000000000000000000000000000000000000000"
            "0000000000000000000000000000000000000000000000000000000000000000"
            "0000000000000000800000000000000002000000000000000500000000000000"
            "68656c6c6f000000000000000000000000000000000000000000000000000000"
            "0000000000000000000000000000000000000000000000000000000000000000"
            "0000000000000000000000000000000000000000000000000000000000000000"
            "0000000000000000000000000000000000000000000000000000000000000000"
            "2a00000000000000200000012000000034120000000000000b00000000000000"
            "3812000000000000785600000000000010000101100000004523000000000000"
            "0c00000000000000100002021000000000000000000000005634000000000000"
            "4000030120000000030000000000000000100000000000000020000000000000"
            "0000000000000000000000000000000067450000000000000d00000011000000"
            "0500000000000000";
        const TString base = HexDecode(hex.data(), hex.size());
        const struct {
            const char* Name;
            size_t Size;
            size_t Offset;
            std::uint16_t Value;
            size_t Width;
            bool Accept;
        } cases[] = {
            {"all_sections", 712, 0, 0, 0, true},
            {"perf_padding", 716, 0, 0, 0, true},
            {"short_header", 167, 0, 0, 0, false},
            {"truncated_payload", 711, 0, 0, 0, false},
            {"wrong_tag", 712, 0, 1, 1, false},
            {"undefined_sample_type", 712, 4, 0, 1, false},
            {"unknown_sample_type", 712, 4, 6, 1, false},
            {"overlapping_sections", 712, 148, 8, 1, false},
            {"gap_between_sections", 712, 148, 24, 1, false},
            {"partial_stack", 712, 146, 15, 1, false},
            {"misaligned_section", 712, 144, 1, 1, false},
            {"section_out_of_bounds", 712, 166, 65535, 2, false},
            {"unknown_language", 712, 554, 4, 1, false},
            {"reserved_language_byte_zero", 712, 555, 0, 1, true},
            {"zero_element_size", 712, 556, 0, 1, false},
            {"misaligned_element_size", 712, 556, 7, 1, false},
            {"zero_payload_size", 712, 552, 0, 1, false},
            {"partial_language_element", 712, 552, 31, 1, false},
            {"truncated_language_header", 712, 166, 1, 1, false},
            {"language_out_of_bounds", 712, 552, 4096, 2, false},
            {"duplicate_language", 712, 594, 0, 1, false},
            {"reserved_language_byte_nonzero", 712, 555, 2, 1, true},
            // Opaque interpreter payloads are accepted here but rejected by Go.
            {"opaque_interpreter_size", 712, 556, 16, 1, true},
            {"reserved_jvm_byte_nonzero", 712, 619, 1, 1, true},
        };
        for (const auto& test : cases) {
            TString record = base;
            record.resize(test.Size, '\x5a');
            for (size_t i = 0; i < test.Width; ++i) {
                record[test.Offset + i] = static_cast<char>(test.Value >> (8 * i));
            }
            bool accepted = false;
            TRawSample sample;
            try {
                sample = TRawSample::Parse(record);
                accepted = true;
            } catch (const yexception&) {
            }
            UNIT_ASSERT_C(accepted == test.Accept, test.Name);
            if (!accepted) {
                continue;
            }
            UNIT_ASSERT_VALUES_EQUAL(sample.Header().cpu, 3);
            UNIT_ASSERT(sample.Header().kthread);
            UNIT_ASSERT_VALUES_EQUAL(sample.Header().runtime, 1000);
            UNIT_ASSERT_VALUES_EQUAL(sample.Header().starttime, 555);
            UNIT_ASSERT_VALUES_EQUAL(sample.KernelStack().Read(1), 0x20);
            UNIT_ASSERT_VALUES_EQUAL(sample.UserStack().Read(0), 0x30);
            UNIT_ASSERT_VALUES_EQUAL(sample.Cgroups().Read(0), 42);
            UNIT_ASSERT_VALUES_EQUAL(sample.Lbr().Count(), 2);
            const auto firstBranch = sample.Lbr().Read(0);
            UNIT_ASSERT_VALUES_EQUAL(firstBranch.from, 0x40);
            UNIT_ASSERT_VALUES_EQUAL(firstBranch.to, 0x50);
            UNIT_ASSERT_VALUES_EQUAL(firstBranch.flags, 7);
            const auto secondBranch = sample.Lbr().Read(1);
            UNIT_ASSERT_VALUES_EQUAL(secondBranch.from, 0x60);
            UNIT_ASSERT_VALUES_EQUAL(secondBranch.to, 0x70);
            UNIT_ASSERT_VALUES_EQUAL(secondBranch.flags, 9);
            UNIT_ASSERT_VALUES_EQUAL(sample.Tls().Count(), 2);
            const auto firstTls = sample.Tls().Read(0);
            UNIT_ASSERT_VALUES_EQUAL(firstTls.offset, 64);
            UNIT_ASSERT(firstTls.type == THREAD_LOCAL_UINT64_TYPE);
            UNIT_ASSERT_VALUES_EQUAL(firstTls.value.number, 99);
            const auto secondTls = sample.Tls().Read(1);
            UNIT_ASSERT_VALUES_EQUAL(secondTls.offset, 128);
            UNIT_ASSERT(secondTls.type == THREAD_LOCAL_STRING_TYPE);
            UNIT_ASSERT_VALUES_EQUAL(secondTls.value.string.len, 5);
            UNIT_ASSERT_VALUES_EQUAL(TStringBuf(secondTls.value.string.string, 5), "hello");
            UNIT_ASSERT_VALUES_EQUAL(sample.LanguageSectionCount(), 4);
            UNIT_ASSERT_VALUES_EQUAL(sample.LanguageSection(LANGUAGE_PYTHON)->Element(0).size(),
                                     TStringBuf(test.Name) == "opaque_interpreter_size" ? 16 : 32);
            UNIT_ASSERT_VALUES_EQUAL(sample.LanguageSection(LANGUAGE_PHP)->Count(), 1);
            UNIT_ASSERT_VALUES_EQUAL(sample.LanguageSection(LANGUAGE_JVM)->Count(), 1);
            UNIT_ASSERT_VALUES_EQUAL(sample.LanguageSection(LANGUAGE_LUA)->Count(), 2);
            UNIT_ASSERT_VALUES_EQUAL(sample.LanguageSection(LANGUAGE_LUA)->Element(0).size(), 32);
        }
    }

    Y_UNIT_TEST(RejectsPartialLbrAndTlsElements) {
        const struct {
            std::uint16_t LbrSize;
            std::uint16_t TlsSize;
            const char* Error;
        } cases[] = {
            {16, 0, "invalid last branch records section alignment/size"},
            {0, 8, "invalid TLS section alignment/size"},
        };
        for (const auto& test : cases) {
            record_sample_header header{};
            header.tag = RECORD_TAG_SAMPLE;
            header.sample_type = SAMPLE_TYPE_PERF_EVENT;
            header.lbr = {.offset = 0, .size = test.LbrSize};
            header.tls = {.offset = test.LbrSize, .size = test.TlsSize};
            const auto dataSize = static_cast<std::uint16_t>(test.LbrSize + test.TlsSize);
            header.cgroups = {.offset = dataSize, .size = 0};
            header.language_sections = {.offset = dataSize, .size = 0};

            // All sections are contiguous and 8-byte aligned; only the
            // selected section's size is not a multiple of its element size.
            TString record(sizeof(header) + dataSize, '\0');
            std::memcpy(record.begin(), &header, sizeof(header));
            UNIT_ASSERT_EXCEPTION_CONTAINS(TRawSample::Parse(record), yexception, test.Error);
        }
    }

    Y_UNIT_TEST(AcceptsValidKthreadRepresentations) {
        for (unsigned value : {0u, 1u}) {
            TString record = MakeRecord();
            record[offsetof(record_sample_header, kthread)] = static_cast<char>(value);
            const auto sample = TRawSample::Parse(record);
            UNIT_ASSERT_VALUES_EQUAL(sample.Header().kthread, value != 0);
        }
    }

    Y_UNIT_TEST(RejectsInvalidKthreadRepresentations) {
        for (unsigned value = 2; value <= 255; ++value) {
            TString record = MakeRecord();
            auto* bytes = reinterpret_cast<unsigned char*>(record.begin());
            bytes[offsetof(record_sample_header, kthread)] = value;
            UNIT_ASSERT_EXCEPTION_CONTAINS(TRawSample::Parse(record), yexception,
                                           "invalid kthread flag");
        }
    }

    Y_UNIT_TEST(BorrowsUnalignedSections) {
        const TString record = MakeRecord();
        const TString storage = TString("x") + record;
        const auto sample = TRawSample::Parse(TStringBuf(storage).SubStr(1));
        UNIT_ASSERT_VALUES_EQUAL(sample.KernelStack().Read(1), 0x20);
        UNIT_ASSERT(sample.KernelStack().Bytes().data() == storage.data() + 1 + sizeof(record_sample_header));
        UNIT_ASSERT(sample.LanguageSection(LANGUAGE_PYTHON)->Element(1).empty());
    }

    Y_UNIT_TEST(AcceptsKernel54FixedSizeRecordWithPerfPadding) {
        TString record = MakeRecord();
        record.resize(sizeof(record_sample_header) + NWire::MaxPackedDataSize, '\0');
        record.append(4, '\x5a');
        const auto sample = TRawSample::Parse(record);
        UNIT_ASSERT_VALUES_EQUAL(sample.UserStack().Read(0), 0x30);
        UNIT_ASSERT_VALUES_EQUAL(sample.Cgroups().Read(0), 42);
        UNIT_ASSERT_VALUES_EQUAL(sample.LanguageSection(LANGUAGE_PYTHON)->Count(), 1);
    }

    Y_UNIT_TEST(AcceptsVariableSizeRecordWithPerfPadding) {
        TString record = MakeRecord();
        record.append(4, '\x5a');
        const auto sample = TRawSample::Parse(record);
        UNIT_ASSERT_VALUES_EQUAL(sample.UserStack().Read(0), 0x30);
        UNIT_ASSERT_VALUES_EQUAL(sample.LanguageSections().size(), PythonSectionSize);
    }

    Y_UNIT_TEST(RejectsSectionExtendingIntoPerfPadding) {
        TString record = MakeRecord();
        record.append(4, '\x5a');
        auto* header = reinterpret_cast<record_sample_header*>(record.begin());
        header->language_sections.size += 4;
        UNIT_ASSERT_EXCEPTION_CONTAINS(TRawSample::Parse(record), yexception,
                                       "language sections section exceeds packed sample");
    }

    Y_UNIT_TEST(RejectsRecordLargerThanMaximumWithPerfPadding) {
        TString record = MakeRecord();
        record.resize(sizeof(record_sample_header) + NWire::MaxPackedDataSize + 5, '\0');
        UNIT_ASSERT_EXCEPTION_CONTAINS(TRawSample::Parse(record), yexception,
                                       "packed sample is too large");
    }

    Y_UNIT_TEST(ParsesValidPackedSample) {
        const TString record = MakeRecord();
        const auto sample = TRawSample::Parse(record);

        UNIT_ASSERT_VALUES_EQUAL(sample.KernelStack().Count(), 2);
        UNIT_ASSERT_VALUES_EQUAL(sample.KernelStack().Read(1), 0x20);
        UNIT_ASSERT_VALUES_EQUAL(sample.UserStack().Read(0), 0x30);
        UNIT_ASSERT_VALUES_EQUAL(sample.Cgroups().Read(0), 42);
        UNIT_ASSERT_VALUES_EQUAL(
            static_cast<std::uint32_t>(sample.Header().sample_type),
            static_cast<std::uint32_t>(SAMPLE_TYPE_PERF_EVENT));
        const auto* language = sample.LanguageSection(LANGUAGE_PYTHON);
        UNIT_ASSERT(language);
        UNIT_ASSERT_VALUES_EQUAL(language->ElementSize(), PythonFrameSize);
        UNIT_ASSERT_VALUES_EQUAL(language->Count(), 1);
        UNIT_ASSERT_VALUES_EQUAL(language->Element(0).size(), PythonFrameSize);
    }

    Y_UNIT_TEST(RejectsTruncatedRecord) {
        TString record = MakeRecord();
        record.resize(record.size() - 1);
        UNIT_ASSERT_EXCEPTION(TRawSample::Parse(record), yexception);
    }

    Y_UNIT_TEST(RejectsNonContiguousSections) {
        TString record = MakeRecord();
        auto* header = reinterpret_cast<record_sample_header*>(record.begin());
        header->user_stack.offset += 8;
        UNIT_ASSERT_EXCEPTION(TRawSample::Parse(record), yexception);
    }

    Y_UNIT_TEST(RejectsPartialElements) {
        TString record = MakeRecord();
        auto* header = reinterpret_cast<record_sample_header*>(record.begin());
        header->kern_stack.size -= 1;
        UNIT_ASSERT_EXCEPTION(TRawSample::Parse(record), yexception);
    }

    Y_UNIT_TEST(RejectsDuplicateLanguage) {
        TString record = MakeRecord();
        auto* header = reinterpret_cast<record_sample_header*>(record.begin());
        header->language_sections.size *= 2;
        const TString duplicate{record.data() + sizeof(record_sample_header) + 32, PythonSectionSize};
        record.append(duplicate);
        UNIT_ASSERT_EXCEPTION(TRawSample::Parse(record), yexception);
    }

    Y_UNIT_TEST(RejectsUnknownLanguage) {
        TString record = MakeRecord();
        auto* language = reinterpret_cast<language_section_header*>(
            record.begin() + sizeof(record_sample_header) + 32);
        language->language = static_cast<language_id>(PERFORATOR_LANGUAGE_COUNT);
        UNIT_ASSERT_EXCEPTION(TRawSample::Parse(record), yexception);
    }

    Y_UNIT_TEST(AcceptsExactMaximumPackedDataSize) {
        record_sample_header header{};
        header.tag = RECORD_TAG_SAMPLE;
        header.sample_type = SAMPLE_TYPE_PERF_EVENT;
        header.kern_stack = {.offset = 0, .size = NWire::MaxPackedDataSize};
        header.user_stack = {.offset = NWire::MaxPackedDataSize, .size = 0};
        header.lbr = {.offset = NWire::MaxPackedDataSize, .size = 0};
        header.tls = {.offset = NWire::MaxPackedDataSize, .size = 0};
        header.cgroups = {.offset = NWire::MaxPackedDataSize, .size = 0};
        header.language_sections = {.offset = NWire::MaxPackedDataSize, .size = 0};

        TString record(sizeof(header) + NWire::MaxPackedDataSize, '\0');
        std::memcpy(record.begin(), &header, sizeof(header));
        const auto sample = TRawSample::Parse(record);
        UNIT_ASSERT_VALUES_EQUAL(sample.KernelStack().Bytes().size(), NWire::MaxPackedDataSize);
        UNIT_ASSERT(sample.LanguageSections().empty());

        record.append(4, '\x5a');
        const auto paddedSample = TRawSample::Parse(record);
        UNIT_ASSERT_VALUES_EQUAL(paddedSample.KernelStack().Bytes().size(), NWire::MaxPackedDataSize);
        UNIT_ASSERT(paddedSample.LanguageSections().empty());

        auto* paddedHeader = reinterpret_cast<record_sample_header*>(record.begin());
        paddedHeader->language_sections.size = 4;
        UNIT_ASSERT_EXCEPTION_CONTAINS(TRawSample::Parse(record), yexception,
                                       "language sections section exceeds packed sample");
    }
} // Y_UNIT_TEST_SUITE(RawSample)

////////////////////////////////////////////////////////////////////////////////

} // anonymous namespace
} // namespace NPerforator::NAgent::NNativePipeline
