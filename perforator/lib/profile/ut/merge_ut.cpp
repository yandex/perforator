#include "golden.h"

#include <perforator/lib/profile/flat_diffable.h>
#include <perforator/lib/profile/merge_manager.h>
#include <perforator/lib/profile/pprof.h>

#include <library/cpp/testing/gtest/gtest.h>
#include <library/cpp/testing/common/env.h>

#include <util/stream/file.h>

using namespace NPerforator::NProfile::NTest;

TEST(MergeProfilesTest, Golden) {
    TVector<NPerforator::NProto::NPProf::Profile> profiles;
    NPerforator::NProto::NPProf::Profile expected;

    for (TFsPath path : NPerforator::NProfile::NTest::ListGoldenProfiles("merge", 11)) {
        TFileInput input{path};

        auto profile = ParsePprof(path);
        if (path.GetName().StartsWith("merged")) {
            expected = std::move(profile);
        } else {
            profiles.emplace_back(std::move(profile));
        }
    }

    Y_ENSURE(profiles.size() > 2);
    Y_ENSURE(expected.sample_size() > 100);

    const ui32 threadCount = 4;
    NPerforator::NProfile::TMergeManager manager{threadCount};

    NPerforator::NProto::NProfile::MergeOptions options;
    options.set_ignore_process_ids(false);
    options.set_ignore_thread_ids(false);
    options.set_cleanup_thread_names(false);
    auto session = manager.StartSession(options);

    for (auto&& pprof : profiles) {
        NPerforator::NProto::NProfile::Profile profile;
        NPerforator::NProfile::ConvertFromPProf(pprof, &profile);
        session->AddProfile(std::move(profile));
    }
    auto merged = std::move(*session).Finish();

    CompareFlatProfiles(expected, merged);
}
