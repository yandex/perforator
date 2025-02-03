#include "profile.h"

#include <util/stream/output.h>

namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

////////////////////////////////////////////////////////////////////////////////

TProfile::TProfile(const NProto::NProfile::Profile* profile)
    : Profile_{profile}
{}

const NProto::NProfile::Metadata& TProfile::GetMetadata() const {
    return Profile_->metadata();
}

const NProto::NProfile::Features& TProfile::GetFeatures() const {
    return Profile_->features();
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile

////////////////////////////////////////////////////////////////////////////////

template <>
void Out<NPerforator::NProfile::TStringRef>(
    IOutputStream& stream,
    const NPerforator::NProfile::TStringRef& ref
) {
    Out<TStringBuf>(stream, ref.View());
}

////////////////////////////////////////////////////////////////////////////////
