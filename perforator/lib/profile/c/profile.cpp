#include "profile.h"

#include "error.hpp"
#include "string.hpp"

#include <perforator/lib/profile/pprof.h>
#include <perforator/proto/pprofprofile/profile.pb.h>
#include <perforator/proto/profile/profile.pb.h>


namespace NPerforator::NProfile::NCWrapper {

NProto::NProfile::Profile* UnwrapProfile(TPerforatorProfile profile) {
    return reinterpret_cast<NProto::NProfile::Profile*>(profile);
}

TPerforatorProfile WrapProfile(NProto::NProfile::Profile profile) {
    return new NProto::NProfile::Profile{std::move(profile)};
}

extern "C" {

void PerforatorProfileDispose(TPerforatorProfile profile) {
    delete UnwrapProfile(profile);
}

TPerforatorError PerforatorProfileParse(const char* ptr, size_t size, TPerforatorProfile* result) {
    return InterceptExceptions([&] {
        auto profile = MakeHolder<NProto::NProfile::Profile>();
        Y_ENSURE(profile->ParseFromArray(ptr, size));
        *result = profile.Release();
    });
}

TPerforatorError PerforatorProfileParsePProf(const char* ptr, size_t size, TPerforatorProfile* result) {
    return InterceptExceptions([&] {
        google::protobuf::Arena arena;
        auto* pprof = google::protobuf::Arena::Create<NProto::NPProf::Profile>(&arena);
        Y_ENSURE(pprof->ParseFromArray(ptr, size));

        auto profile = MakeHolder<NProto::NProfile::Profile>();
        ConvertFromPProf(*pprof, profile.Get());
        *result = profile.Release();
    });
}

TPerforatorError PerforatorProfileSerialize(TPerforatorProfile profile, TPerforatorString* result) {
    return InterceptExceptions([&] {
        TString str = UnwrapProfile(profile)->SerializeAsStringOrThrow();
        *result = MakeString(std::move(str));
    });
}

TPerforatorError PerforatorProfileSerializePProf(TPerforatorProfile profile, TPerforatorString* result) {
    return InterceptExceptions([&] {
        NProto::NPProf::Profile pprof;
        ConvertToPProf(*UnwrapProfile(profile), &pprof);
        *result = MakeString(pprof.SerializeAsStringOrThrow());
    });
}

} // extern "C"

} // namespace NPerforator::NProfile::NCWrapper
