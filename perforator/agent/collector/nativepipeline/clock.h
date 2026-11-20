#pragma once

#include <util/system/types.h>

#include <time.h>

namespace NPerforator::NAgent::NNativePipeline {

// Calls require serialization. Calibration is refreshed at most once a second
// during conversion; failed refreshes retain the last successful offset.
class TMonotonicClockConverter {
public:
    using TClockReader = int (*)(clockid_t, timespec*);

    explicit TMonotonicClockConverter(TClockReader readClock = clock_gettime);

    i64 MonotonicToUnixNanos(ui64 collectionTime);
    ui64 UpdateErrors() const noexcept;

private:
    bool UpdateDelta() noexcept;

    TClockReader ReadClock_;
    i64 DeltaNanos_ = 0;
    i64 NextUpdateNanos_ = 0;
    ui64 UpdateErrors_ = 0;
};

} // namespace NPerforator::NAgent::NNativePipeline
