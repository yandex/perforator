#include <perforator/lib/profile/flat_diffable.h>
#include <perforator/lib/profile/merge_manager.h>
#include <perforator/lib/profile/pprof.h>
#include <perforator/lib/profile/ut/lib/golden.h>

#include <library/cpp/testing/gtest/gtest.h>
#include <library/cpp/testing/common/env.h>

#include <util/stream/file.h>

using namespace NPerforator::NProfile::NTest;

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

