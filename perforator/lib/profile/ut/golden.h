#pragma once

#include <perforator/lib/profile/flat_diffable.h>

#include <library/cpp/testing/common/env.h>
#include <library/cpp/testing/gtest/gtest.h>

#include <util/generic/maybe.h>


namespace NPerforator::NProfile::NTest {

NProto::NPProf::Profile ParsePprof(const TFsPath& path);

TVector<TFsPath> ListGoldenProfiles(TStringBuf pattern, TMaybe<size_t> expectedProfileCount = Nothing());

TString SerializeFlatProfile(const TFlatDiffableProfile& profile);

TMap<TString, ui64> CountFlatProfileEvents(const TFlatDiffableProfile& profile);

template <typename L, typename R>
void CompareFlatProfiles(const L& lhs, const R& rhs) {
    // Our profiles are somewhat malformed
    TFlatDiffableProfileOptions options{
        .LabelBlacklist = {"comm"},
    };
    TFlatDiffableProfile left{lhs, options};
    TFlatDiffableProfile right{rhs, options};

    auto lhsEvents = CountFlatProfileEvents(left);
    auto rhsEvents = CountFlatProfileEvents(right);
    EXPECT_EQ(lhsEvents, rhsEvents);

    TString pprofString = SerializeFlatProfile(left);
    TString protoString = SerializeFlatProfile(right);
    EXPECT_EQ(pprofString, protoString);
}

} // namespace NPerforator::NProfile::NTest
