#include <perforator/lib/profile/builder.h>
#include <perforator/lib/profile/flat_diffable.h>
#include <perforator/lib/profile/merge.h>
#include <perforator/lib/profile/merge_manager.h>
#include <perforator/lib/profile/pprof.h>
#include <perforator/lib/profile/profile.h>
#include <perforator/lib/profile/ut/lib/golden.h>

#include <library/cpp/testing/gtest/gtest.h>
#include <library/cpp/testing/common/env.h>

#include <util/stream/file.h>

#include <initializer_list>

using namespace NPerforator::NProfile::NTest;

namespace {

struct TValueTypeSpec {
    TStringBuf Type;
    TStringBuf Unit;
    ui64 Value = 0;
};

NPerforator::NProto::NProfile::Profile MakeProfile(
    std::initializer_list<TValueTypeSpec> valueTypes,
    TStringBuf defaultSampleType = {}
) {
    TVector<TValueTypeSpec> specs{valueTypes.begin(), valueTypes.end()};

    NPerforator::NProto::NProfile::Profile profile;
    NPerforator::NProfile::TProfileBuilder builder{&profile};

    TVector<NPerforator::NProfile::TValueTypeId> valueTypeIds;
    valueTypeIds.reserve(specs.size());
    for (const auto& valueType : specs) {
        valueTypeIds.push_back(builder.AddValueType(valueType.Type, valueType.Unit));
    }

    if (!defaultSampleType.empty()) {
        builder.Metadata().GetProto().set_default_sample_type(*builder.AddString(defaultSampleType));
    }

    auto sampleKey = builder.AddSampleKey(NPerforator::NProfile::TSampleKeyInfo{});
    auto sample = builder.AddSample();
    sample.SetSampleKey(sampleKey);
    for (size_t i = 0; i < specs.size(); ++i) {
        sample.AddValue(valueTypeIds[i], specs[i].Value);
    }
    sample.Finish();

    std::move(builder).Finish();
    return profile;
}

// Builds a single cpu.cycles sample carrying the given string labels and value.
NPerforator::NProto::NProfile::Profile MakeLabeledProfile(
    std::initializer_list<std::pair<TStringBuf, TStringBuf>> labels,
    ui64 value
) {
    NPerforator::NProto::NProfile::Profile profile;
    NPerforator::NProfile::TProfileBuilder builder{&profile};

    auto valueTypeId = builder.AddValueType("cpu", "cycles");
    builder.Metadata().GetProto().set_default_sample_type(*builder.AddString("cpu"));

    auto keyBuilder = builder.AddSampleKey();
    for (auto&& [key, val] : labels) {
        keyBuilder.AddLabel(builder.AddStringLabel(key, val));
    }
    auto sampleKey = keyBuilder.Finish();

    auto sample = builder.AddSample();
    sample.SetSampleKey(sampleKey);
    sample.AddValue(valueTypeId, value);
    sample.Finish();

    std::move(builder).Finish();
    return profile;
}

TStringBuf GetDefaultSampleType(const NPerforator::NProto::NProfile::Profile& profile) {
    NPerforator::NProfile::TProfile view{&profile};
    return view.String(NPerforator::NProfile::TStringId::FromInternalIndex(profile.metadata().default_sample_type())).View();
}

i32 FindValueType(
    const NPerforator::NProto::NProfile::Profile& profile,
    TStringBuf type,
    TStringBuf unit
) {
    NPerforator::NProfile::TProfile view{&profile};
    for (i32 i = 0; i < profile.samples().values_size(); ++i) {
        auto valueType = view.ValueType(NPerforator::NProfile::TValueTypeId::FromInternalIndex(i));
        if (valueType.GetType().View() == type && valueType.GetUnit().View() == unit) {
            return i;
        }
    }
    return -1;
}

ui64 GetSampleValue(
    const NPerforator::NProto::NProfile::Profile& profile,
    TStringBuf type,
    TStringBuf unit
) {
    i32 idx = FindValueType(profile, type, unit);
    Y_ENSURE(idx >= 0);
    return profile.samples().values(idx).value(0);
}

} // anonymous namespace

TEST(MergeProfilesTest, Golden) {
    TVector<TString> profilesBytes;
    NPerforator::NProto::NPProf::Profile expected;

    for (TFsPath path : NPerforator::NProfile::NTest::ListGoldenProfiles(SRC_("testprofiles/merge"), "[^\\.]*(.[0-9]+)?.pb.gz", 11)) {
        TFileInput input{path};

        auto profileBytes = DecompressPprof(path);
        if (path.GetName().StartsWith("merged")) {
            expected.ParseFromStringOrThrow(profileBytes);
        } else {
            profilesBytes.emplace_back(std::move(profileBytes));
        }
    }

    Y_ENSURE(profilesBytes.size() > 2);
    Y_ENSURE(expected.sample_size() > 100);

    const ui32 threadCount = 4;
    NPerforator::NProfile::TMergeManager manager{threadCount};

    // This golden fixture predates the strip/sanitize defaults flipping to true; pin them
    // off so the test exercises pure merge/dedup against the committed golden. The default-on
    // behavior is covered by the yandex-specific render/merge tests.
    NPerforator::NProto::NProfile::MergeOptions opts;
    opts.set_strip_garbage_root_frames(false);
    opts.set_sanitize_thread_names(false);
    auto session = manager.StartSession(opts);

    for (auto&& profileBytes : profilesBytes) {
        NPerforator::NProto::NProfile::Profile profile;
        NPerforator::NProfile::ConvertFromPProf(profileBytes, &profile);
        session->AddProfile(std::move(profile));
    }
    auto merged = std::move(*session).Finish();

    CompareFlatProfiles(expected, merged);
}

TEST(MergeProfilesTest, MergesValueTypeUnion) {
    auto cpu = MakeProfile({{"cpu", "cycles", 10}}, "cpu");
    auto wall = MakeProfile({{"wall", "seconds", 20}}, "wall");

    NPerforator::NProto::NProfile::Profile merged;
    NPerforator::NProfile::MergeProfiles({cpu, wall}, &merged);

    ASSERT_EQ(merged.samples().key_size(), 1);
    ASSERT_EQ(merged.samples().values_size(), 2);
    EXPECT_EQ(GetSampleValue(merged, "cpu", "cycles"), 10);
    EXPECT_EQ(GetSampleValue(merged, "wall", "seconds"), 20);
    EXPECT_EQ(GetDefaultSampleType(merged), "cpu");
}

TEST(MergeProfilesTest, FiltersValueTypesAndDefaultSampleType) {
    auto profile = MakeProfile({
        {"cpu", "cycles", 10},
        {"wall", "seconds", 20},
    }, "wall");

    NPerforator::NProto::NProfile::MergeOptions keepDefault;
    keepDefault.mutable_value_type_filter()->add_allowlist("wall.seconds");
    NPerforator::NProto::NProfile::Profile preserved;
    NPerforator::NProfile::MergeProfiles({profile}, &preserved, keepDefault);
    ASSERT_EQ(preserved.samples().values_size(), 1);
    EXPECT_EQ(GetSampleValue(preserved, "wall", "seconds"), 20);
    EXPECT_EQ(GetDefaultSampleType(preserved), "wall");

    NPerforator::NProto::NProfile::MergeOptions dropDefault;
    dropDefault.mutable_value_type_filter()->add_allowlist("cpu.cycles");
    NPerforator::NProto::NProfile::Profile dropped;
    NPerforator::NProfile::MergeProfiles({profile}, &dropped, dropDefault);
    ASSERT_EQ(dropped.samples().values_size(), 1);
    EXPECT_EQ(GetSampleValue(dropped, "cpu", "cycles"), 10);
    EXPECT_EQ(dropped.metadata().default_sample_type(), 0);
}

// A sample filter keyed on a label that keys_show strips must still match: the parallel
// merger used to re-run the filter at the combine stage on already-stripped intermediate
// profiles and drop every sample but worker 0's. With many profiles per worker, the loss
// is large (and can empty the profile entirely).
TEST(MergeProfilesTest, ParallelMergerKeepsFilteredSamplesAcrossWorkers) {
    constexpr size_t profileCount = 16;
    constexpr ui64 perProfileValue = 10;

    NPerforator::NProto::NProfile::MergeOptions options;
    (*options.mutable_sample_filter()->mutable_required_all_of_string_labels())["tls:perforator_tls_ui64_TestID"] =
        "1586802";
    options.mutable_label_filter()->add_keys_show("comm");

    NPerforator::NProfile::TMergeManager manager{/*threadCount=*/4};
    auto session = manager.StartSession(options);
    for (size_t i = 0; i < profileCount; ++i) {
        session->AddProfile(MakeLabeledProfile({
            {"tls:perforator_tls_ui64_TestID", "1586802"},
            {"comm", "worker"},
        }, perProfileValue));
    }
    auto merged = std::move(*session).Finish();

    ASSERT_EQ(merged.samples().key_size(), 1);
    EXPECT_EQ(GetSampleValue(merged, "cpu", "cycles"), perProfileValue * profileCount);
}

TEST(MergeProfilesTest, ParallelMergerIgnoresEmptyWorkers) {
    auto profile = MakeProfile({{"cpu", "cycles", 10}}, "cpu");

    NPerforator::NProfile::TMergeManager manager{4};
    NPerforator::NProto::NProfile::MergeOptions options;
    auto session = manager.StartSession(options);
    session->AddProfile(std::move(profile));
    auto merged = std::move(*session).Finish();

    ASSERT_EQ(merged.samples().key_size(), 1);
    ASSERT_EQ(merged.samples().values_size(), 1);
    EXPECT_EQ(GetDefaultSampleType(merged), "cpu");
}
