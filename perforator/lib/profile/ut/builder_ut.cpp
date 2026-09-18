#include <perforator/lib/profile/builder.h>
#include <perforator/lib/profile/profile.h>

#include <library/cpp/testing/gtest/gtest.h>

#include <absl/hash/hash_testing.h>

#include <util/random/random.h>
#include <util/generic/function_ref.h>


using namespace NPerforator::NProfile;

namespace {

struct TRandomIndex {
    template <CStrongIndex Index>
    operator Index() const {
        return Index::FromInternalIndex(RandomNumber<ui32>(Max<i32>()));
    }
} R;

template <typename F>
testing::AssertionResult VerifyTypeImplementsAbslHashCorrectly(F&& factory) {
    using T = decltype(factory());

    TVector<T> values;
    for (int i = 0; i < 1000; ++i) {
        values.push_back(factory());
    }

    return absl::VerifyTypeImplementsAbslHashCorrectly(values);
}

} // anonymous namespace

TEST(AbslHashes, ValueType) {
    EXPECT_TRUE(VerifyTypeImplementsAbslHashCorrectly([] {
        return TValueTypeInfo{R, R};
    }));
}

TEST(ProfileBuilder, PreservesSampleTimestamps) {
    const TSampleTimestamp timestamps[] = {
        {10, 900'000'000}, // Start timestamp, zero delta.
        {10, 950'000'000}, // Positive delta within the same second.
        {11, 0},          // Exactly on the next second boundary.
        {11, 100'000'000}, // Carry nanoseconds into seconds.
        {13, 1},          // Delta spanning multiple seconds.
        {10, 800'000'000}, // Negative delta within the same second.
        {10, 0},          // Negative delta ending on a second boundary.
        {9, 999'999'999},  // Negative delta crossing a second boundary.
        {8, 900'000'000},  // Negative whole-second delta.
        {7, 100'000'000},  // Negative delta spanning multiple seconds.
    };

    NPerforator::NProto::NProfile::Profile proto;
    TProfileBuilder builder{&proto};
    auto key = builder.AddSampleKey(TSampleKeyInfo{});
    for (const auto& ts : timestamps) {
        builder.AddSample().SetSampleKey(key).SetTimestamp(ts.Seconds, ts.NanoSeconds).Finish();
    }
    std::move(builder).Finish();

    TProfile profile{&proto};
    ui32 index = 0;
    for (const auto& expected : timestamps) {
        SCOPED_TRACE(index);
        auto sample = profile.Sample(TSampleId::FromInternalIndex(index++));
        auto actual = sample.GetProtoTimestamp();
        ASSERT_TRUE(actual.Defined());
        EXPECT_EQ(actual->seconds(), expected.Seconds);
        EXPECT_EQ(actual->nanos(), expected.NanoSeconds);
        auto instant = sample.GetInstantTimestamp();
        ASSERT_TRUE(instant.Defined());
        EXPECT_EQ(instant->MicroSeconds(), expected.Seconds * 1'000'000 + expected.NanoSeconds / 1000);
    }
}

TEST(ProfileBuilder, PreservesMissingSampleTimestamp) {
    NPerforator::NProto::NProfile::Profile proto;
    TProfileBuilder builder{&proto};
    auto key = builder.AddSampleKey(TSampleKeyInfo{});
    builder.AddSample().SetSampleKey(key).Finish();
    std::move(builder).Finish();

    auto sample = TProfile{&proto}.Sample(TSampleId::FromInternalIndex(0));
    EXPECT_FALSE(sample.GetProtoTimestamp().Defined());
    EXPECT_FALSE(sample.GetInstantTimestamp().Defined());
}
