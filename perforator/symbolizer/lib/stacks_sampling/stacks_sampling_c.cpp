#include "stacks_sampling_c.h"

#include <perforator/symbolizer/lib/stacks_sampling/aggregating_sampler.hpp>

#include <perforator/proto/pprofprofile/profile.pb.h>

#include <memory>

namespace {

NPerforator::NStacksSampling::TAggregatingSampler* FromOpaque(void* sampler) {
    return reinterpret_cast<NPerforator::NStacksSampling::TAggregatingSampler*>(sampler);
}

}

void *CreateAggregatingStacksSampler(ui64 rate) {
    auto samplerPtr = std::make_unique<NPerforator::NStacksSampling::TAggregatingSampler>(rate);

    return samplerPtr.release();
}

void DestroyAggregatingStacksSampler(void* sampler) {
    std::unique_ptr<NPerforator::NStacksSampling::TAggregatingSampler> samplerPtr{FromOpaque(sampler)};

    samplerPtr.reset();
}

void AddProfileIntoAggregatingStacksSampler(
    void* sampler,
    const char* profileBytes,
    ui64 profileBytesLen
) {
    auto* samplerPtr = FromOpaque(sampler);
    samplerPtr->AddProfile({profileBytes, profileBytesLen});
}

void ExtractResultingProfileFromSampler(
    void* sampler,
    const char** dataPtr,
    ui64* dataLenPtr,
    ui64* isEmpty
) {
    if (dataPtr == nullptr || dataLenPtr == nullptr || isEmpty == nullptr) {
        return;
    }

    auto* samplerPtr = FromOpaque(sampler);

    const auto& profile = samplerPtr->GetResultingProfile();
    *isEmpty = (profile.sampleSize() == 0 ? 1UL : 0UL);
    if (*isEmpty) {
        *dataPtr = nullptr;
        *dataLenPtr = 0;
        return;
    }

    const auto profileBytes = profile.SerializeAsString();
    const auto profileBytesCount = profileBytes.size();

    char* buffer = new char[profileBytesCount];
    if (buffer == nullptr) {
        return;
    }
    memcpy(buffer, profileBytes.data(), profileBytesCount);

    *dataPtr = buffer;
    *dataLenPtr = profileBytesCount;
}
