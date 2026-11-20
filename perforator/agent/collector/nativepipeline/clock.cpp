#include "clock.h"

#include <util/generic/yexception.h>
#include <util/system/error.h>

namespace NPerforator::NAgent::NNativePipeline {
namespace {

constexpr i64 NanosPerSecond = 1'000'000'000;

i64 ToNanos(const timespec& value) noexcept {
    return value.tv_sec * NanosPerSecond + value.tv_nsec;
}

} // namespace

TMonotonicClockConverter::TMonotonicClockConverter(TClockReader readClock)
    : ReadClock_{readClock}
{
    Y_ENSURE(ReadClock_, "expected a clock reader");
    Y_ENSURE(UpdateDelta(), "failed to initialize native clock offset: " << LastSystemErrorText());
}

bool TMonotonicClockConverter::UpdateDelta() noexcept {
    timespec realtime{}, monotonic{};
    if (ReadClock_(CLOCK_REALTIME, &realtime) != 0 || ReadClock_(CLOCK_MONOTONIC, &monotonic) != 0) {
        return false;
    }
    DeltaNanos_ = ToNanos(realtime) - ToNanos(monotonic);
    NextUpdateNanos_ = ToNanos(monotonic) + NanosPerSecond;
    return true;
}

i64 TMonotonicClockConverter::MonotonicToUnixNanos(ui64 collectionTime) {
    timespec monotonic{};
    if (ReadClock_(CLOCK_MONOTONIC, &monotonic) != 0) {
        ++UpdateErrors_;
    } else if (const auto now = ToNanos(monotonic); now >= NextUpdateNanos_) {
        NextUpdateNanos_ = now + NanosPerSecond;
        if (!UpdateDelta()) {
            ++UpdateErrors_;
        }
    }
    return static_cast<i64>(collectionTime + static_cast<ui64>(DeltaNanos_));
}

ui64 TMonotonicClockConverter::UpdateErrors() const noexcept {
    return UpdateErrors_;
}

} // namespace NPerforator::NAgent::NNativePipeline
