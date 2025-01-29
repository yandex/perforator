#include <perforator/symbolizer/lib/autofdo/autofdo_c.h>
#include <perforator/symbolizer/lib/autofdo/autofdo_input_builder.h>

#include <cstring>
#include <memory>

#include <fmt/format.h>

namespace {

NPerforator::NAutofdo::TBatchInputBuilder* FromOpaque(void* builder) {
    return reinterpret_cast<NPerforator::NAutofdo::TBatchInputBuilder*>(builder);
}

NPerforator::NAutofdo::TBatchBuildIdGuesser* FromOpaqueGuesser(void* guesser) {
    return reinterpret_cast<NPerforator::NAutofdo::TBatchBuildIdGuesser*>(guesser);
}

}

extern "C" {

void* MakeBatchBuilder(ui64 buildersCount, const char* buildId) {
    auto builderPtr = std::make_unique<NPerforator::NAutofdo::TBatchInputBuilder>(buildersCount, buildId);

    return builderPtr.release();
}

void DestroyBatchBuilder(void* builder) {
    std::unique_ptr<NPerforator::NAutofdo::TBatchInputBuilder> builderPtr{FromOpaque(builder)};

    builderPtr.reset();
}

void AddProfile(void* builder, ui64 builderIndex, const char* profileBytes, ui64 profileBytesLen) {
    auto* builderPtr = FromOpaque(builder);

    builderPtr->GetBuilder(builderIndex).AddProfile({profileBytes, profileBytesLen});
}

void Finalize(
    void* builder,
    ui64* totalProfiles,
    ui64* totalBranches, ui64* totalSamples, ui64* bogusLbrEntries,
    ui64* branchCountMapSize, ui64* rangeCountMapSize, ui64* addressCountMapSize,
    char** autofdoInput, char** boltInput) {
    auto* builderPtr = FromOpaque(builder);

    const auto autofdoInputData = std::move(*builderPtr).Finalize();
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

    const auto autofdoInputStr = SerializeAutofdoInput(autofdoInputData);
    const auto boltInputStr = SerializeAutofdoInputInBoltPreaggregatedFormat(autofdoInputData);
    const auto assignStrTo = [] (char** destination, const std::string& source) {
        if (destination != nullptr) {
            *destination = strndup(source.data(), source.size());
        }
    };
    assignStrTo(autofdoInput, autofdoInputStr);
    assignStrTo(boltInput, boltInputStr);
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
