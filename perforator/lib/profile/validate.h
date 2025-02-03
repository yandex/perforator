#pragma once

#include <util/generic/yexception.h>


namespace NPerforator::NProto::NProfile {

////////////////////////////////////////////////////////////////////////////////

class Profile;

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProto::NProfile

namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

struct TProfileValidationOptions {
    // Check that all indices inside profile are point to valid locations.
    // If enabled, the profile will be full-scanned.
    bool CheckIndices = false;
};

void ValidateProfile(
    const NProto::NProfile::Profile& profile,
    TProfileValidationOptions options = {}
);

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
