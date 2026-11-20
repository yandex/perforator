#pragma once

#include "clock.h"
#include "raw_sample.h"
#include "runtime.hpp"
#include "state.hpp"

#include <perforator/lib/profile/builder.h>

#include <util/generic/array_ref.h>
#include <util/generic/string.h>
#include <util/generic/vector.h>

namespace NPerforator::NAgent::NNativePipeline {

// Selection is made by routing; wall adds a value to the event profile.
enum class EProfileOutput {
    Event,
    LastBranchRecords,
};

struct TProfileOutput {
    EProfileOutput Kind = EProfileOutput::Event;
    bool IncludeWallTime = false;
    TString UprobeSampleKind;
};

// Builder and clock are borrowed and must outlive the accumulator. Samples
// must be written only through this accumulator. Finish finalizes the caller's proto;
// the next interval requires a new builder/accumulator pair.
// All operations, including access to the shared clock, require serialization.
class TSampleAccumulator {
public:
    TSampleAccumulator(
        NProfile::TProfileBuilder& builder,
        TMonotonicClockConverter& clock,
        TProfileOutput output = {},
        TProfileOptions options = {},
        bool emitPerSampleTimestamps = false
    );
    TSampleAccumulator(const TSampleAccumulator&) = delete;
    TSampleAccumulator& operator=(const TSampleAccumulator&) = delete;

    // Returns whether a nonempty sample was written. Resolution errors throw
    // before writing; unknown events are skipped. Allocation failure during
    // writing requires discarding this accumulator and its builder.
    bool Accumulate(
        const TStateSnapshot& state,
        IRuntimeResolver& resolver,
        const TRawSample& sample
    );

    void Finish();

private:
    struct TValueType {
        TString Kind;
        TString Unit;

        bool operator==(const TValueType&) const = default;
    };

    struct TSampleValue {
        TValueType Type;
        ui64 Value;
    };

    TVector<TSampleValue> BuildSampleValues(const record_sample_header& header, const TStateSnapshot& state) const;
    void CheckValueTypes(TConstArrayRef<TSampleValue> values) const;
    void InitializeValueTypes(TConstArrayRef<TSampleValue> values);
    void FeedTimestamp(i64 timestampNanos);
    void AddCommonLabels(
        NProfile::TProfileBuilder::TSimpleSampleKeyBuilder& sampleKeyBuilder,
        const TStateSnapshot& state,
        const TProcessState* process,
        const TRawSample& sample
    );

    TVector<TValueType> ValueTypes_;
    TVector<NProfile::TValueTypeId> ValueTypeIds_;
    const TProfileOutput Output_;
    const TProfileOptions Options_;
    bool EmitPerSampleTimestamps_ = false;
    NProfile::TProfileBuilder& Builder_;
    TMonotonicClockConverter& Clock_;
    bool HasTimestampRange_ = false;
    i64 MinTimestampNanos_ = 0;
    i64 MaxTimestampNanos_ = 0;
    bool Finished_ = false;
};

} // namespace NPerforator::NAgent::NNativePipeline
