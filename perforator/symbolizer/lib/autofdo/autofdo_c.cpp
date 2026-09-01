#include <perforator/symbolizer/lib/autofdo/autofdo_c.h>
#include <perforator/symbolizer/lib/autofdo/autofdo_input_builder.h>

#include <perforator/lib/profile/c/error.hpp>

#include <cstdlib>
#include <cstring>
#include <memory>
#include <new>

namespace {

NPerforator::NAutofdo::TBatchInputBuilder* FromOpaque(void* builder) {
    return reinterpret_cast<NPerforator::NAutofdo::TBatchInputBuilder*>(builder);
}

NPerforator::NAutofdo::TBatchBuildIdGuesser* FromOpaqueGuesser(void* guesser) {
    return reinterpret_cast<NPerforator::NAutofdo::TBatchBuildIdGuesser*>(guesser);
}

}

extern "C" {

void* MakeBatchBuilder(ui64 buildersCount, const char* buildId, const char* binaryPath, TPerforatorError* error) {
    void* result = nullptr;
    auto capturedError = NPerforator::NProfile::NCWrapper::InterceptExceptions([&] {
        auto builderPtr = std::make_unique<NPerforator::NAutofdo::TBatchInputBuilder>(buildersCount, buildId, binaryPath);
        result = builderPtr.release();
    });
    if (error != nullptr) {
        *error = capturedError;
    } else {
        PerforatorErrorDispose(capturedError);
    }
    return result;
}

void DestroyBatchBuilder(void* builder) {
    std::unique_ptr<NPerforator::NAutofdo::TBatchInputBuilder> builderPtr{FromOpaque(builder)};

    builderPtr.reset();
}

TPerforatorError AddProfile(
    void* builder,
    ui64 builderIndex,
    const char* serviceName,
    const char* profileBytes,
    ui64 profileBytesLen
) {
    return NPerforator::NProfile::NCWrapper::InterceptExceptions([&] {
        auto* builderPtr = FromOpaque(builder);
        builderPtr->GetBuilder(builderIndex).AddProfile(
            {serviceName},
            {profileBytes, profileBytesLen});
    });
}

TPerforatorError Finalize(
    void* builder,
    const char* binaryPath,
    ui64* totalProfiles,
    ui64* totalBranches, ui64* totalSamples, ui64* bogusLbrEntries,
    ui64* branchCountMapSize, ui64* rangeCountMapSize, ui64* addressCountMapSize,
    //
    ui64* profilesByServiceMapLen,
    const char*** profilesByServiceMapServices,
    ui64** profilesByServiceMapCounts,
    //
    const char** autofdoInput, const char** boltInput) {
    return NPerforator::NProfile::NCWrapper::InterceptExceptions([&] {
        auto* builderPtr = FromOpaque(builder);
        const auto autofdoInputData = std::move(*builderPtr).Finalize();
        const auto [autofdoInputStr, boltInputStr] = SerializePGOInputsForBinary(autofdoInputData, binaryPath);

        const auto assignTo = [] (ui64* destination, ui64 source) {
            if (destination != nullptr) {
                *destination = source;
            }
        };
        assignTo(totalProfiles, autofdoInputData.Meta.TotalProfiles);
        assignTo(totalBranches, autofdoInputData.Meta.TotalBranches);
        assignTo(totalSamples, autofdoInputData.Meta.TotalSamples);
        assignTo(bogusLbrEntries, autofdoInputData.Meta.BogusLbrEntries);
        assignTo(branchCountMapSize, autofdoInputData.BranchCountMap.size());
        assignTo(rangeCountMapSize, autofdoInputData.RangeCountMap.size());
        assignTo(addressCountMapSize, autofdoInputData.AddressCountMap.size());

        const auto assignStrTo = [] (const char** destination, const std::string& source) {
            if (destination != nullptr) {
                *destination = strndup(source.data(), source.size());
            }
        };
        assignStrTo(autofdoInput, autofdoInputStr);
        assignStrTo(boltInput, boltInputStr);

        if (profilesByServiceMapLen != nullptr &&
            profilesByServiceMapServices != nullptr &&
            profilesByServiceMapCounts != nullptr) {
            const auto& profilesCountByService = autofdoInputData.Meta.ProfilesCountByService;

            *profilesByServiceMapLen = profilesCountByService.size();
            auto services = static_cast<const char**>(std::malloc(sizeof(const char*) * *profilesByServiceMapLen));
            auto counts = static_cast<ui64*>(std::malloc(sizeof(ui64) * *profilesByServiceMapLen));
            if (*profilesByServiceMapLen != 0 && (services == nullptr || counts == nullptr)) {
                std::free(services);
                std::free(counts);
                throw std::bad_alloc{};
            }
            *profilesByServiceMapServices = services;
            *profilesByServiceMapCounts = counts;

            ui64 idx = 0;
            for (const auto& [service, count] : profilesCountByService) {
                services[idx] = strndup(service.data(), service.size());
                counts[idx] = count;
                ++idx;
            }
        }
    });
}

ui64 GetBinaryExecutableBytes(const char* binaryPath) {
    return NPerforator::NAutofdo::GetBinaryInstructionsBytesSize(binaryPath);
}

///////////////////////////////////////////////////////////////////////////////////////////

void* MakeBatchBuildIdGuesser(ui64 guessersCount) {
    auto guesserPtr = std::make_unique<NPerforator::NAutofdo::TBatchBuildIdGuesser>(guessersCount);

    return guesserPtr.release();
}

void DestroyBatchBuildIdGuesser(void* guesser) {
    std::unique_ptr<NPerforator::NAutofdo::TBatchBuildIdGuesser> guesserPtr{FromOpaqueGuesser(guesser)};

    guesserPtr.reset();
}

void FeedProfileIntoGuesser(void* guesser, ui64 guesserIndex, const char* profileBytes, ui64 profileBytesLen) {
    auto* guesserPtr = FromOpaqueGuesser(guesser);

    guesserPtr->GetGuesser(guesserIndex).FeedProfile({profileBytes, profileBytesLen});
}

const char* TryGuessBuildID(void* guesser) {
    const auto* guesserPtr = FromOpaqueGuesser(guesser);

    const auto buildIdOpt = guesserPtr->GuessBuildID();
    if (!buildIdOpt.has_value() || buildIdOpt->size() == 0) {
        return nullptr;
    }

    return strndup(buildIdOpt->data(), buildIdOpt->size());
}

}
