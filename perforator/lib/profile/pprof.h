#pragma once

#include <perforator/proto/pprofprofile/profile.pb.h>
#include <perforator/proto/profile/profile.pb.h>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

void ConvertFromPProf(const NProto::NPProf::Profile& from, NProto::NProfile::Profile* to);

void ConvertFromPProf2(const NProto::NPProf::Profile& from, NProto::NProfile::Profile* to);

void ConvertToPProf(const NProto::NProfile::Profile& from, NProto::NPProf::Profile* to);

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile

