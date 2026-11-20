#include "signals.h"

#include <util/generic/scope.h>
#include <util/generic/yexception.h>

#include <cstdlib>

extern "C" char* PerforatorGoSignalName(int32_t) __attribute__((weak));

namespace NPerforator::NAgent::NNativePipeline {

TString GetSignalName(i32 signal) {
    Y_ENSURE(PerforatorGoSignalName, "Go signal name callback is not linked");
    char* name = PerforatorGoSignalName(signal);
    Y_DEFER {
        std::free(name);
    };
    return name ? TString{name} : TString{};
}

} // namespace NPerforator::NAgent::NNativePipeline
