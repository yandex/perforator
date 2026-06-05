#include "profile.h"

#include <util/generic/string.h>
#include <util/generic/vector.h>
#include <util/stream/output.h>

namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

TProfile::TProfile(const NProto::NProfile::Profile* profile)
    : Profile_{profile}
{}

const NProto::NProfile::Metadata& TProfile::GetMetadata() const {
    return Profile_->metadata();
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile

////////////////////////////////////////////////////////////////////////////////

template <>
void Out<NPerforator::NProfile::TProfileString>(
    IOutputStream& stream,
    const NPerforator::NProfile::TProfileString& ref
) {
    Out<TStringBuf>(stream, ref.View());
}

////////////////////////////////////////////////////////////////////////////////
