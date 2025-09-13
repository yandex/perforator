#pragma once

#include <perforator/proto/profile/profile.pb.h>
#include <perforator/proto/profile/merge_options.pb.h>

#include <util/generic/array_ref.h>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

// NB: @TProfileMerger is single-threaded and not thread-safe.
class TProfileMerger {
public:
    TProfileMerger(
        NProto::NProfile::Profile* merged,
        const NProto::NProfile::MergeOptions& options
    );

    TProfileMerger(const TProfileMerger& rhs) = delete;
    TProfileMerger(TProfileMerger&& rhs) noexcept;
    TProfileMerger& operator=(const TProfileMerger& rhs) = delete;
    TProfileMerger& operator=(TProfileMerger&& rhs) noexcept;

    ~TProfileMerger();

    // Merge one profile into the resulting one.
    // This function is not thread safe.
    void Add(const NProto::NProfile::Profile& profile);

    // Finalizes the merge process, performing any last calculations or
    // data consolidation.
    //
    // This method is &&-qualified, meaning it consumes the TProfileMerger
    // instance. This design enforces a single-use lifecycle where Finish()
    // is the terminal operation. After calling Finish(), the merger instance
    // is in a moved-from state and must not be used again.
    //
    // For convenience, it returns the pointer to the `merged` profile that was
    // provided in the constructor.
    //
    // This function is not thread safe.
    NProto::NProfile::Profile* Finish() &&;

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
    const NProto::NProfile::MergeOptions& options = {}
);

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
