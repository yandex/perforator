#pragma once

#include "profile.h"

#include <perforator/proto/profile/profile.pb.h>

#include <util/generic/array_ref.h>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

struct TMergeOptions {
    // Do not aggregate samples with same stacks from different processes into one.
    // If disabled, merged profile will not have per-process information.
    bool KeepProcesses = true;

    // Do not merge stacks based on binary addresses, use symbolized names instead.
    // If false, merged profile will not contain per-binary info.
    // Greatly reduces profile size that spans multiple versions of similar binaries.
    // Does not work if profiles are not symbolized.
    bool KeepBinaries = true;

    // Keep binaries. If false, then samples from the same binary (build id),
    // but with different binary paths will be merged together.
    bool KeepBinaryPaths = true;

    // Keep timing profiles.
    // If disabled, the resulting profile samples will not contain timestamps.
    // This allows us to merge different samples together, reducing profile size.
    bool KeepTimestamps = false;

    // Keep exact source code locations (column and line numbers).
    bool KeepLineNumbers = true;

    // If enabled, normalize some well-known value type units.
    // For example, profiles with value types [wall.nanoseconds] and [wall.us]
    // can be merged to a profile of value type [wall.microseconds].
    //
    // In addition, this option allows merger to select some value type unit
    // that is different from the types of the provided profiles, probably
    // losing some precision. For example, if we are merging 1024 profiles of
    // value type [wall.nanoseconds] and each profile contains sample with
    // value of 2^60, than we cannot represent such value in a merged profile.
    // Without this option, the merging process will fail.
    // With this option the merger can select common value type [wall.second].
    bool NormalizeValueTypes = true;

    // Sanitize thread & process names. In particular, remove digits from names.
    // Many threadpools set their worker thread names as "ThreadPoolName-WorkerIndex".
    bool CleanupThreadNames = true;

    // Allows to filter profile labels. By default, sample labels are stored
    // as-is in the merged profile. If source profiles contain many different
    // labels, for example, trace ids, it can blow up the merged profile size.
    //
    // The filter must return true for labels that pass this filter.
    // Such labels will be kept in the merged profile.
    std::function<bool(TLabel)> LabelFilter;
};

// NB: @TProfileMerger is not thread-safe.
class TProfileMerger {
public:
    TProfileMerger(NProto::NProfile::Profile* merged, TMergeOptions options);

    ~TProfileMerger();

    // Merge one profile into the resulting one.
    // This function is not thread safe.
    void Add(const NProto::NProfile::Profile& profile);

    // Do some bookkeeping work to finish merging.
    // You must call TProfileMerger::Finish() after TProfileMerger::Add().
    void Finish();

private:
    class TImpl;
    THolder<TImpl> Impl_;
};

////////////////////////////////////////////////////////////////////////////////

// Convenience function for a small number of profiles. Prefer to use
// TProfileMerger directly to save memory: you do not have to keep all
// the profiles in RAM.
void MergeProfiles(
    TConstArrayRef<NProto::NProfile::Profile> profiles,
    NProto::NProfile::Profile* merged,
    TMergeOptions options = {}
);

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
