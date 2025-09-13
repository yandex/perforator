#pragma once

#include "string.h"

#include <util/generic/string.h>
#include <util/generic/yexception.h>


namespace NPerforator::NProfile::NCWrapper {

[[nodiscard]] TPerforatorString MakeString(TString str);

} // namespace NPerforator::NProfile::NCWrapper
