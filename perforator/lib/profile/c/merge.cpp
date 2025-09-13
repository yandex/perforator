#include "merge.h"
#include "error.hpp"
#include "profile.hpp"

#include <perforator/lib/profile/merge.h>
#include <perforator/lib/profile/merge_manager.h>

#include <util/generic/yexception.h>
#include <util/thread/pool.h>


namespace NPerforator::NProfile::NCWrapper {

namespace {

TMergeManager* UnwrapManager(TPerforatorProfileMergeManager manager) {
    return reinterpret_cast<TMergeManager*>(manager);
}

TMergeSession* UnwrapSession(TPerforatorProfileMerger session) {
    return reinterpret_cast<TMergeSession*>(session);
}

} // anonymous namespace

extern "C" {

TPerforatorError PerforatorMakeMergeManager(
    int threadCount,
    TPerforatorProfileMergeManager* result
) {
    return InterceptExceptions([&] {
        *result = new TMergeManager(threadCount);
    });
}

void PerforatorDestroyMergeManager(
    TPerforatorProfileMergeManager manager
) {
    delete UnwrapManager(manager);
}

TPerforatorError PerforatorMergerStart(
    TPerforatorProfileMergeManager manager,
    const char* protoOptionsBegin,
    size_t protoOptionsSize,
    TPerforatorProfileMerger* result
) {
    return InterceptExceptions([&] {
        NProto::NProfile::MergeOptions options;
        Y_ENSURE(options.ParseFromArray(protoOptionsBegin, protoOptionsSize));
        *result = UnwrapManager(manager)->StartSession(options).Release();
    });
}

TPerforatorError PerforatorMergerAddProfile(
    TPerforatorProfileMerger merger,
    TPerforatorProfile profile
) {
    return InterceptExceptions([&] {
        UnwrapSession(merger)->AddProfile(*UnwrapProfile(profile));
    });
}

TPerforatorError PerforatorMergerFinish(
    TPerforatorProfileMerger merger,
    TPerforatorProfile* result
) {
    return InterceptExceptions([&] {
        auto profile = std::move(*UnwrapSession(merger)).Finish();
        *result = WrapProfile(std::move(profile));
    });
}

void PerforatorMergerDispose(
    TPerforatorProfileMerger merger
) {
    delete UnwrapSession(merger);
}

} // extern "C"

} // namespace NPerforator::NProfile::NCWrapper
