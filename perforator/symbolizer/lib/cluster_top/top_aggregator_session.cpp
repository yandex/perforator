#include "top_aggregator_session.h"

#include <perforator/lib/profile/validate.h>

namespace NPerforator::NClusterTop {

void TTopAggregatorSession::InitializeSymbolizer(
    TArrayRef<const char> buildId,
    TArrayRef<const char> gsymPath) {
    Aggregator_.InitializeSymbolizer(buildId, gsymPath);
}

void TTopAggregatorSession::AddYaprofBytes(TArrayRef<const char> bytes) {
    if (bytes.data() == nullptr || bytes.size() == 0) {
        return;
    }

    Profile_.Clear();
    Y_ENSURE(Profile_.ParseFromArray(bytes.data(), bytes.size()));
    NProfile::ValidateProfile(Profile_, {.CheckIndices = true});
    Aggregator_.AddProfile(Profile_);
}

void TTopAggregatorSession::MergeAggregator(const TTopAggregatorSession& other) {
    Aggregator_.MergeAggregator(other.Aggregator_);
}

TPerfTopAggregator::PerfTop TTopAggregatorSession::ExtractEntries() {
    return Aggregator_.ExtractEntries();
}

} // namespace NPerforator::NClusterTop
