#include "golden.h"

#include <perforator/lib/profile/flat_diffable.h>
#include <perforator/lib/profile/pprof.h>
#include <perforator/lib/profile/profile.h>

#include <library/cpp/testing/gtest/gtest.h>
#include <library/cpp/testing/common/env.h>


using namespace NPerforator::NProfile::NTest;

// NOLINTNEXTLINE(readability-identifier-naming)
struct GoldenProfileTest : testing::TestWithParam<TFsPath> {};

TEST_P(GoldenProfileTest, ConvertPprofCanon) {
    NPerforator::NProto::NPProf::Profile pprofProto = ParsePprof(GetParam());

    // Make new protobuf profile from pprof
    NPerforator::NProto::NProfile::Profile profileProto;
    NPerforator::NProfile::ConvertFromPProf(pprofProto, &profileProto);

    CompareFlatProfiles(pprofProto, profileProto);
}

TEST_P(GoldenProfileTest, ConvertPprofRoundTripCanon) {
    NPerforator::NProto::NPProf::Profile pprofOriginalProto = ParsePprof(GetParam());

    // Make new protobuf profile from pprof
    NPerforator::NProto::NProfile::Profile profileProto;
    NPerforator::NProfile::ConvertFromPProf(pprofOriginalProto, &profileProto);

    NPerforator::NProto::NPProf::Profile pprofConvertedProto;
    NPerforator::NProfile::ConvertToPProf(profileProto, &pprofConvertedProto);

    CompareFlatProfiles(pprofOriginalProto, pprofConvertedProto);
}

INSTANTIATE_TEST_SUITE_P(
    GoldenProfiles,
    GoldenProfileTest,
    testing::ValuesIn(NPerforator::NProfile::NTest::ListGoldenProfiles("diff", 5))
);
