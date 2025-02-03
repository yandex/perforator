#include <perforator/lib/profile/builder.h>

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
