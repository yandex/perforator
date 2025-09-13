#pragma once

#include <perforator/proto/profile/profile.pb.h>
#include <perforator/proto/profile/merge_options.pb.h>

#include <util/thread/pool.h>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

struct TParallelProfileMergerOptions {
    // Specifies which parts of the profile should be kept in the merged profile.
    NProto::NProfile::MergeOptions MergeOptions;

    // Number of parallel merges allowed to run.
    // Increases CPU and memory usage, reduces wall time.
    // NB: Merger spawns @ConcurrencyLevel tasks in the provided thread pool.
    // If the thread pool does not have sufficient capacity,
    // the merge will perform slower.
    ui32 ConcurrencyLevel = 10;

    // Limit on the number of pending profiles in queue.
    // If you add more than BufferSize profiles.
    // If BufferSize is zero, the queue is unlimited.
    ui32 BufferSize = 32;
};

class TParallelProfileMerger {
public:
    TParallelProfileMerger(
        NProto::NProfile::Profile* merged,
        TParallelProfileMergerOptions options,
        IThreadPool* pool
    );

    ~TParallelProfileMerger();

    // Merge one profile into the resulting one.
    // This function is thread safe, but may block.
    void Add(NProto::NProfile::Profile profile);

    // Do some bookkeeping work to finish merging.
    // You must call Finish() or Abort() after Add().
    void Finish() &&;

    // Abort merging. Discard merged profiles.
    void Abort() &&;

private:
    class TImpl;
    THolder<TImpl> Impl_;
};

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
