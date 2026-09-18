#include <perforator/lib/profile/builder.h>
#include <perforator/lib/profile/pprof.h>
#include <perforator/lib/profile/profile.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <util/generic/yexception.h>
#include <util/stream/mem.h>
#include <util/stream/zlib.h>

TEST(PProfParserTest, RejectsInvalidGzipData) {
    // Gzip magic bytes followed by garbage (not valid gzip stream)
    const char gzipData[] = "\x1f\x8b\x08\x00\x00\x00\x00\x00\x00\x03some garbage";
    TStringBuf input(gzipData, sizeof(gzipData) - 1);

    NPerforator::NProto::NProfile::Profile profile;

    // Should throw when trying to decompress invalid gzip
    EXPECT_THROW(
        NPerforator::NProfile::ConvertFromPProf(input, &profile),
        yexception
    );
}

TEST(PProfParserTest, RejectsRandomGarbage) {
    // Random bytes that don't form valid protobuf
    const char garbage[] = "\xff\xff\xff\xff\xff\xff\xff\xff";
    TStringBuf input(garbage, sizeof(garbage) - 1);

    NPerforator::NProto::NProfile::Profile profile;

    EXPECT_THROW_MESSAGE_HAS_SUBSTR(
        NPerforator::NProfile::ConvertFromPProf(input, &profile),
        yexception,
        "not a valid protobuf"
    );
}

TEST(PProfParserTest, AcceptsEmptyProfile) {
    // Empty protobuf is valid (all fields optional)
    TStringBuf input;

    NPerforator::NProto::NProfile::Profile profile;

    // Should not throw
    EXPECT_NO_THROW(NPerforator::NProfile::ConvertFromPProf(input, &profile));
}

TEST(PProfParserTest, AcceptsGzipCompressedEmptyProfile) {
    // Create gzip-compressed empty data (valid empty protobuf)
    TString compressed;
    {
        TStringOutput output(compressed);
        TZLibCompress compressor(&output, ZLib::GZip);
        // Write nothing - empty protobuf
        compressor.Finish();
    }

    NPerforator::NProto::NProfile::Profile profile;

    // Should not throw - valid gzip with empty profile inside
    EXPECT_NO_THROW(NPerforator::NProfile::ConvertFromPProf(compressed, &profile));
}

TEST(PProfConverterTest, RestoresAgentNumericLabelUnits) {
    using namespace NPerforator::NProfile;
    struct TExpectedLabel {
        TStringBuf Key;
        i64 Value;
        TStringBuf Unit;
    };
    const TExpectedLabel labels[] = {
        {"pid", 0, "pid"},
        {"tid", 2, "tid"},
        {"innermost_pidns_pid", 3, "id"},
        {"innermost_pidns_tid", 4, "id"},
        {"absolute_timestamp", 1'800'000'000'123'456'789LL, "ns"},
        {"custom", -5, ""},
        {"", 0, ""},
    };

    for (bool existingUnits : {false, true}) {
        SCOPED_TRACE(existingUnits);
        NPerforator::NProto::NProfile::Profile profile;
        TProfileBuilder builder{&profile};
        if (existingUnits) {
            builder.AddString("id");
            builder.AddString("ns");
        }
        auto keyBuilder = builder.AddSampleKey();
        auto groupBuilder = builder.AddLabelGroup();
        for (const auto& label : labels) {
            auto id = builder.AddNumericLabel(label.Key, label.Value);
            // Exercise both direct and grouped labels, including each unit.
            if (label.Key == "pid" || label.Key == "innermost_pidns_pid") {
                groupBuilder.AddLabel(id);
            } else {
                keyBuilder.AddLabel(id);
            }
        }
        keyBuilder.SetLabelGroup(groupBuilder.Finish());
        // A well-known key used as a string must not acquire a numeric unit.
        keyBuilder.AddLabel(builder.AddStringLabel("absolute_timestamp", "text"));
        auto key = keyBuilder.Finish();
        builder.AddSample().SetSampleKey(key).SetTimestamp(10, 0).Finish();
        builder.AddSample().SetSampleKey(key).SetTimestamp(11, 0).Finish();
        std::move(builder).Finish();

        for (bool viaBytes : {false, true}) {
            SCOPED_TRACE(viaBytes);
            NPerforator::NProto::NPProf::Profile converted;
            if (viaBytes) {
                TString bytes;
                ConvertToPProf(profile, &bytes);
                ASSERT_TRUE(converted.ParseFromString(bytes));
            } else {
                ConvertToPProf(profile, &converted);
            }
            const TProfile source{&profile};
            ASSERT_EQ(converted.string_table_size(), source.Strings().size() + (existingUnits ? 0 : 2));
            for (auto str : source.Strings()) {
                EXPECT_EQ(converted.string_table(*str.GetIndex()), str.View());
            }
            ASSERT_EQ(converted.sample_size(), 2);
            for (const auto& sample : converted.sample()) {
                ASSERT_EQ(sample.label_size(), std::size(labels) + 1);
                for (const auto& label : sample.label()) {
                    ASSERT_GE(label.key(), 0);
                    ASSERT_LT(label.key(), converted.string_table_size());
                    ASSERT_GE(label.num_unit(), 0);
                    ASSERT_LT(label.num_unit(), converted.string_table_size());
                }
                for (const auto& expected : labels) {
                    SCOPED_TRACE(expected.Key);
                    int found = 0;
                    for (const auto& label : sample.label()) {
                        if (label.str() == 0 && converted.string_table(label.key()) == expected.Key) {
                            ++found;
                            EXPECT_EQ(label.num(), expected.Value);
                            EXPECT_EQ(converted.string_table(label.num_unit()), expected.Unit);
                        }
                    }
                    EXPECT_EQ(found, 1);
                }
                int stringLabels = 0;
                for (const auto& label : sample.label()) {
                    if (label.str() != 0) {
                        ++stringLabels;
                        ASSERT_LT(label.str(), converted.string_table_size());
                        EXPECT_EQ(converted.string_table(label.key()), "absolute_timestamp");
                        EXPECT_EQ(converted.string_table(label.str()), "text");
                        EXPECT_EQ(label.num_unit(), 0);
                    }
                }
                EXPECT_EQ(stringLabels, 1);
            }
            // Both converters must reuse existing strings and add each missing unit once.
            for (TStringBuf unit : {"id", "ns"}) {
                int count = 0;
                for (const auto& str : converted.string_table()) {
                    count += str == unit;
                }
                EXPECT_EQ(count, 1);
            }
        }
    }
}

TEST(PProfConverterTest, PreservesDuplicateStringIndices) {
    using namespace NPerforator::NProfile;
    NPerforator::NProto::NProfile::Profile profile;
    TProfileBuilder builder{&profile};
    auto key = builder.AddSimpleSampleKey()
        .AddLabel(builder.AddNumericLabel("pid", 7))
        .Finish();
    builder.AddSample().SetSampleKey(key).Finish();
    std::move(builder).Finish();

    // The builder deduplicates strings; construct an input referencing a duplicate.
    auto* numbers = profile.mutable_labels()->mutable_numbers();
    const int labelIndex = numbers->key_size() - 1;
    const ui32 originalIndex = numbers->key(labelIndex);
    auto* strtab = profile.mutable_strtab();
    const int duplicateIndex = strtab->offset_size();
    strtab->add_offset(strtab->offset(originalIndex));
    strtab->add_length(strtab->length(originalIndex));
    numbers->set_key(labelIndex, duplicateIndex);

    for (bool viaBytes : {false, true}) {
        SCOPED_TRACE(viaBytes);
        NPerforator::NProto::NPProf::Profile converted;
        if (viaBytes) {
            TString bytes;
            ConvertToPProf(profile, &bytes);
            ASSERT_TRUE(converted.ParseFromString(bytes));
            // The native parser must also accept a string table after samples.
            NPerforator::NProto::NProfile::Profile restored;
            ConvertFromPProf(bytes, &restored);
            NPerforator::NProto::NPProf::Profile roundTrip;
            ConvertToPProf(restored, &roundTrip);
            ASSERT_EQ(roundTrip.sample_size(), 1);
            ASSERT_EQ(roundTrip.sample(0).label_size(), 1);
            const auto& label = roundTrip.sample(0).label(0);
            EXPECT_EQ(roundTrip.string_table(label.key()), "pid");
            EXPECT_EQ(label.num(), 7);
        } else {
            ConvertToPProf(profile, &converted);
        }
        const TProfile source{&profile};
        ASSERT_EQ(converted.string_table_size(), source.Strings().size());
        for (auto str : source.Strings()) {
            EXPECT_EQ(converted.string_table(*str.GetIndex()), str.View());
        }
        ASSERT_EQ(converted.sample_size(), 1);
        ASSERT_EQ(converted.sample(0).label_size(), 1);
        EXPECT_EQ(converted.sample(0).label(0).key(), duplicateIndex);
        EXPECT_EQ(converted.sample(0).label(0).num_unit(), originalIndex);
    }
}

TEST(PProfConverterTest, DoesNotAddUnusedNumericLabelUnits) {
    using namespace NPerforator::NProfile;
    NPerforator::NProto::NProfile::Profile profile;
    TProfileBuilder builder{&profile};
    auto key = builder.AddSimpleSampleKey()
        .AddLabel(builder.AddNumericLabel("custom", 42))
        .AddLabel(builder.AddStringLabel("absolute_timestamp", "text"))
        .Finish();
    builder.AddSample().SetSampleKey(key).Finish();
    std::move(builder).Finish();

    for (bool viaBytes : {false, true}) {
        SCOPED_TRACE(viaBytes);
        NPerforator::NProto::NPProf::Profile converted;
        if (viaBytes) {
            TString bytes;
            ConvertToPProf(profile, &bytes);
            ASSERT_TRUE(converted.ParseFromString(bytes));
        } else {
            ConvertToPProf(profile, &converted);
        }
        for (const auto& str : converted.string_table()) {
            EXPECT_NE(str, "id");
            EXPECT_NE(str, "ns");
        }
        ASSERT_EQ(converted.sample_size(), 1);
        ASSERT_EQ(converted.sample(0).label_size(), 2);
        for (const auto& label : converted.sample(0).label()) {
            EXPECT_EQ(label.num_unit(), 0);
        }
    }
}
