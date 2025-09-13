#include "cluster_top_c.h"

#include <memory>

#include <perforator/symbolizer/lib/cluster_top/service_perf_top_aggregator.h>

namespace {

NPerforator::NClusterTop::TServicePerfTopAggregator* FromOpaque(void *aggregator) {
    return reinterpret_cast<NPerforator::NClusterTop::TServicePerfTopAggregator*>(aggregator);
}

constexpr std::size_t kBigIntSizeof = 16;

}

extern "C" {

void *MakeServicePerfTopAggregator() {
    auto aggregatorPtr = std::make_unique<NPerforator::NClusterTop::TServicePerfTopAggregator>();

    return aggregatorPtr.release();
}

void DestroyServicePerfTopAggregator(void *aggregator) {
    std::unique_ptr<NPerforator::NClusterTop::TServicePerfTopAggregator> aggregatorPtr{FromOpaque(aggregator)};

    aggregatorPtr.reset();
}

void InitializeSymbolizerForServicePerfTopAggregator(
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

void AddProfileIntoServicePerfTopAggregator(
    void *aggregator,
    const char* service,
    ui64 serviceLen,
    const char* profileBytes,
    ui64 profileBytesLen
) {
    auto *aggregatorPtr = FromOpaque(aggregator);

    aggregatorPtr->AddProfile({service, serviceLen}, {profileBytes, profileBytesLen});
}

void MergeServicePerfTopAggregators(void *aggregator, void *otherAggregator) {
    auto* dst = FromOpaque(aggregator);
    const auto* src = FromOpaque(otherAggregator);

    dst->MergeAggregator(*src);
}

void FinalizeServicePerfTopAggregator(
    void *aggregator,
    ui64* nEntries,
    const char*** functions,
    char** selfCycles,
    char** cumulativeCycles
) {
    if (nEntries == nullptr || functions == nullptr ||
        selfCycles == nullptr || cumulativeCycles == nullptr) {
        return;
    }

    const auto top = FromOpaque(aggregator)->ExtractEntries();

    const auto valuesCount = top.Functions.size();
    *nEntries = valuesCount;
    *functions = new const char*[valuesCount];
    *selfCycles = new char[valuesCount * kBigIntSizeof];
    *cumulativeCycles = new char[valuesCount * kBigIntSizeof];

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
        (*functions)[i] = ::strndup(function.Name.data(), function.Name.size());

        serializeUint128(*selfCycles + i * kBigIntSizeof, function.SelfCycles);
        serializeUint128(*cumulativeCycles + i * kBigIntSizeof, function.CumulativeCycles);
    }
}

}
