#include "cluster_top_c.h"

#include <memory>
#include <perforator/lib/profile/c/error.hpp>

#include <perforator/symbolizer/lib/cluster_top/top_aggregator_session.h>

namespace {

NPerforator::NClusterTop::TTopAggregatorSession* FromOpaque(void *aggregator) {
    return reinterpret_cast<NPerforator::NClusterTop::TTopAggregatorSession*>(aggregator);
}

}

extern "C" {

void *MakePerfTopAggregator() {
    auto aggregatorPtr = std::make_unique<NPerforator::NClusterTop::TTopAggregatorSession>();

    return aggregatorPtr.release();
}

void DestroyPerfTopAggregator(void *aggregator) {
    std::unique_ptr<NPerforator::NClusterTop::TTopAggregatorSession> aggregatorPtr{FromOpaque(aggregator)};

    aggregatorPtr.reset();
}

void InitializeSymbolizerForPerfTopAggregator(
    void *aggregator,
    const char* buildIdBytes, ui64 buildIdBytesLen,
    const char* gsymPathBytes, ui64 gsymPathBytesLen
) {
    auto *aggregatorPtr = FromOpaque(aggregator);

    aggregatorPtr->InitializeSymbolizer(
        {buildIdBytes, buildIdBytesLen},
        {gsymPathBytes, gsymPathBytesLen}
    );
}

TPerforatorError AddYaprofIntoPerfTopAggregator(
    void *aggregator,
    const char* profileBytes,
    ui64 profileBytesLen
) {
    return NPerforator::NProfile::NCWrapper::InterceptExceptions([&] {
        FromOpaque(aggregator)->AddYaprofBytes({profileBytes, profileBytesLen});
    });
}

void MergePerfTopAggregators(void *aggregator, void *otherAggregator) {
    auto* dst = FromOpaque(aggregator);
    const auto* src = FromOpaque(otherAggregator);

    dst->MergeAggregator(*src);
}

void FinalizePerfTopAggregator(
    void *aggregator,
    ui64* nEntries,
    TClusterTopFunction** functions
) {
    if (nEntries == nullptr || functions == nullptr) {
        return;
    }

    const auto top = FromOpaque(aggregator)->ExtractEntries();

    const auto valuesCount = top.Functions.size();
    *nEntries = valuesCount;
    *functions = new TClusterTopFunction[valuesCount];

    const auto serializeUint128 = [](char* buf, const ui128& value) {
        const auto serializeInBigEndian = [](char *buf, ui64 value) {
            for (std::size_t i = 0; i < 8; ++i) {
                buf[8 - 1 - i] = value % 256;
                value /= 256;
            }
        };

        serializeInBigEndian(buf, GetHigh(value));
        serializeInBigEndian(buf + 8, GetLow(value));
    };

    for (std::size_t i = 0; i < valuesCount; ++i) {
        const auto& function = top.Functions[i];
        auto& output = (*functions)[i];
        output.Name = ::strndup(function.Name.data(), function.Name.size());
        output.BuildId = ::strndup(function.BuildId.data(), function.BuildId.size());
        output.SourceFile = ::strndup(function.SourceFile.data(), function.SourceFile.size());
        output.SourceLine = function.SourceLine;
        output.SourceColumn = function.SourceColumn;
        serializeUint128(output.SelfCycles, function.SelfCycles);
        serializeUint128(output.CumulativeCycles, function.CumulativeCycles);
    }
}

void FreePerfTopAggregatorResult(ui64 nEntries, TClusterTopFunction* functions) {
    for (ui64 i = 0; i < nEntries; ++i) {
        free(functions[i].Name);
        free(functions[i].BuildId);
        free(functions[i].SourceFile);
    }
    delete[] functions;
}

}
