#include "merge.h"
#include "parallel_merge.h"

#include <library/cpp/threading/blocking_queue/blocking_queue.h>
#include <library/cpp/threading/future/future.h>
#include <library/cpp/threading/future/async.h>
#include <library/cpp/threading/future/wait/wait.h>

#include <util/datetime/base.h>
#include <util/generic/deque.h>
#include <util/generic/hash_set.h>
#include <util/generic/queue.h>


namespace NPerforator::NProfile {

class TParallelProfileMerger::TImpl {
    struct TMergeResult {
        ui32 WorkerIndex = 0;
        TProfileMerger UnfinishedMerger;
    };

public:
    TImpl(
        NProto::NProfile::Profile* merged,
        TParallelProfileMergerOptions options,
        IThreadPool* pool
    );

    ~TImpl();

    void Add(NProto::NProfile::Profile&& profile);

    void Finish() &&;

    void Abort() &&;

private:
    TMergeResult WorkerThread(ui32 workerId);
    TMergeResult CombineMergers(TMergeResult&& lhs, TMergeResult&& rhs) const;
    NThreading::TFuture<void> SetupMergingPipeline();

private:
    NProto::NProfile::Profile* Merged_;
    TParallelProfileMergerOptions Options_;
    IThreadPool* Pool_;
    NThreading::TBlockingQueue<NProto::NProfile::Profile> PendingProfiles_;

    TDeque<NProto::NProfile::Profile> IntermediateProfileStorage_;
    TVector<NProto::NProfile::Profile*> IntermediateProfiles_;

    NThreading::TFuture<void> MergerFuture_;
};

TParallelProfileMerger::TImpl::TImpl(
    NProto::NProfile::Profile* merged,
    TParallelProfileMergerOptions options,
    IThreadPool* pool
)
    : Merged_(merged)
    , Options_(options)
    , Pool_(pool)
    , PendingProfiles_(Options_.BufferSize)
{
    IntermediateProfiles_.push_back(Merged_);
    for (ui32 i = 1; i < Options_.ConcurrencyLevel; ++i) {
        IntermediateProfiles_.push_back(&IntermediateProfileStorage_.emplace_back());
    }

    MergerFuture_ = SetupMergingPipeline();
}

TParallelProfileMerger::TImpl::~TImpl() {
    std::move(*this).Abort();
}

void TParallelProfileMerger::TImpl::Add(NProto::NProfile::Profile&& profile) {
    PendingProfiles_.Push(std::move(profile));
}

void TParallelProfileMerger::TImpl::Finish() && {
    PendingProfiles_.Stop();
    MergerFuture_.GetValueSync();
}

void TParallelProfileMerger::TImpl::Abort() && {
    if (MergerFuture_.Initialized()) {
        PendingProfiles_.Stop();
        MergerFuture_.Wait();
    }
}

NThreading::TFuture<void> TParallelProfileMerger::TImpl::SetupMergingPipeline() {
    TQueue<NThreading::TFuture<TMergeResult>> mergers;
    for (ui32 i = 0; i < Options_.ConcurrencyLevel; ++i) {
        mergers.push(NThreading::Async([this, i] {
            return WorkerThread(i);
        }, *Pool_));
    }

    while (mergers.size() > 1) {
        auto lhs = std::move(mergers.front());
        mergers.pop();
        auto rhs = std::move(mergers.front());
        mergers.pop();

        mergers.push(NThreading::WaitAll(lhs.IgnoreResult(), rhs.IgnoreResult()).Apply([
            this,
            lhs = std::move(lhs),
            rhs = std::move(rhs)
        ](const NThreading::TFuture<void>&) mutable {
            return CombineMergers(lhs.ExtractValue(), rhs.ExtractValue());
        }));
    }

    Y_ABORT_UNLESS(mergers.size() == 1);
    return std::move(mergers.front()).Apply([](NThreading::TFuture<TMergeResult> f) {
        TMergeResult result = f.ExtractValue();
        Y_ABORT_UNLESS(result.WorkerIndex == 0);
        std::move(result.UnfinishedMerger).Finish();
    });
}

TParallelProfileMerger::TImpl::TMergeResult TParallelProfileMerger::TImpl::WorkerThread(ui32 workerId) {
    TProfileMerger merger{IntermediateProfiles_.at(workerId), Options_.MergeOptions};

    while (auto maybeProfile = PendingProfiles_.Pop()) {
        merger.Add(maybeProfile.GetRef());
    }

    return {workerId, std::move(merger)};
}

TParallelProfileMerger::TImpl::TMergeResult TParallelProfileMerger::TImpl::CombineMergers(
    TParallelProfileMerger::TImpl::TMergeResult&& lhs,
    TParallelProfileMerger::TImpl::TMergeResult&& rhs
) const {
    // We need to preserve mergers order to guarantee that the final merger
    // writes into the @Merged_ profile.
    if (lhs.WorkerIndex > rhs.WorkerIndex) {
        return CombineMergers(std::move(rhs), std::move(lhs));
    }

    NProto::NProfile::Profile* profile = std::move(rhs.UnfinishedMerger).Finish();
    Y_ABORT_UNLESS(profile);

    lhs.UnfinishedMerger.Add(*profile);
    *profile = {};

    return std::move(lhs);
}

TParallelProfileMerger::TParallelProfileMerger(
    NProto::NProfile::Profile* merged,
    TParallelProfileMergerOptions options,
    IThreadPool* pool
)
    : Impl_(new TImpl(merged, options, pool))
{}

TParallelProfileMerger::~TParallelProfileMerger() = default;

void TParallelProfileMerger::Add(NProto::NProfile::Profile profile) {
    Impl_->Add(std::move(profile));
}

void TParallelProfileMerger::Finish() && {
    std::move(*Impl_).Finish();
}

void TParallelProfileMerger::Abort() && {
    std::move(*Impl_).Abort();
}

} // namespace NPerforator::NProfile
