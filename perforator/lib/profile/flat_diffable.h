#pragma once

#include "profile.h"
#include "util/generic/function_ref.h"

#include <perforator/proto/pprofprofile/profile.pb.h>
#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/int128/int128.h>

#include <util/generic/hash_set.h>
#include <util/generic/map.h>


namespace NPerforator::NProfile {

struct TFlatDiffableProfileOptions {
    bool PrintTimestamps = true;
    bool PrintAddresses = true;
    bool PrintBuildIds = true;
    THashSet<TString> LabelBlacklist;
};

class TFlatDiffableProfile {
public:
    TFlatDiffableProfile(const NProto::NPProf::Profile& profile, TFlatDiffableProfileOptions options = {});
    TFlatDiffableProfile(const NProto::NProfile::Profile& profile, TFlatDiffableProfileOptions options = {});
    TFlatDiffableProfile(TProfile profile, TFlatDiffableProfileOptions options = {});

    void IterateSamples(TFunctionRef<void(TStringBuf key, const TMap<TString, ui64>& values)> consumer) const;
    void WriteTo(IOutputStream& out) const;

private:
    TMap<TString, TMap<TString, ui64>> Samples_;
};

} // namespace NPerforator::NProfile
