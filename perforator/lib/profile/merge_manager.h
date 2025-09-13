#pragma once

#include "parallel_merge.h"


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

class TMergeSession : TNonCopyable {
public:
    TMergeSession(TParallelProfileMergerOptions options, IThreadPool* pool);

    void AddProfile(NProto::NProfile::Profile profile);

    NProto::NProfile::Profile Finish() &&;

private:
    NProto::NProfile::Profile Profile_;
    TParallelProfileMerger Merger_;
};

////////////////////////////////////////////////////////////////////////////////

class TMergeManager {
public:
    TMergeManager(ui32 threadCount);

    THolder<TMergeSession> StartSession(NProto::NProfile::MergeOptions options);

private:
    const ui32 ThreadCount_;
    THolder<IThreadPool> Pool_;
};

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
