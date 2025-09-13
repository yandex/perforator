#include "merge_manager.h"


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

TMergeSession::TMergeSession(
    TParallelProfileMergerOptions options,
    IThreadPool* pool
)
    : Merger_{&Profile_, std::move(options), pool}
{}

void TMergeSession::AddProfile(NProto::NProfile::Profile profile) {
    Merger_.Add(std::move(profile));
}

NProto::NProfile::Profile TMergeSession::Finish() && {
    std::move(Merger_).Finish();
    return std::move(Profile_);
}

////////////////////////////////////////////////////////////////////////////////

TMergeManager::TMergeManager(ui32 threadCount)
    : ThreadCount_{threadCount}
    , Pool_{CreateThreadPool(threadCount)}
{}

THolder<TMergeSession> TMergeManager::StartSession(NProto::NProfile::MergeOptions options) {
    return THolder{new TMergeSession{{
        .MergeOptions = std::move(options),
        .ConcurrencyLevel = ThreadCount_,
        .BufferSize = ThreadCount_ * 2,
    }, Pool_.Get()}};
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
