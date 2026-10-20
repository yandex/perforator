#pragma once

#include <perforator/symbolizer/lib/cluster_top/perf_top_aggregator.h>

#include <util/generic/array_ref.h>

namespace NPerforator::NClusterTop {

// C ABI handle: owns reusable yaprof storage and the per-worker accumulator.
// C++ callers with a parsed profile can use TPerfTopAggregator directly.
class TTopAggregatorSession final {
public:
    void InitializeSymbolizer(TArrayRef<const char> buildId, TArrayRef<const char> gsymPath);

    void AddYaprofBytes(TArrayRef<const char> bytes);

    void MergeAggregator(const TTopAggregatorSession& other);

    TPerfTopAggregator::PerfTop ExtractEntries();

private:
    // Reused across profiles; the aggregator copies names into its own storage.
    NProto::NProfile::Profile Profile_;
    TPerfTopAggregator Aggregator_;
};

} // namespace NPerforator::NClusterTop
