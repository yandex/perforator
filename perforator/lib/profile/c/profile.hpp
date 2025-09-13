#pragma once

#include "profile.h"

#include <perforator/proto/profile/profile.pb.h>

#include <util/generic/string.h>
#include <util/generic/yexception.h>


namespace NPerforator::NProfile::NCWrapper {

[[nodiscard]] NProto::NProfile::Profile* UnwrapProfile(TPerforatorProfile profile);
[[nodiscard]] TPerforatorProfile WrapProfile(NProto::NProfile::Profile profile);

} // namespace NPerforator::NProfile::NCWrapper
