#pragma once

#include <util/generic/string.h>
#include <util/system/types.h>

namespace NPerforator::NAgent::NNativePipeline {

// Returns the Go signal name, or an empty string for an unknown signal.
TString GetSignalName(i32 signal);

} // namespace NPerforator::NAgent::NNativePipeline
