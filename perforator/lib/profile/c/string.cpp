#include "string.hpp"


namespace NPerforator::NProfile::NCWrapper {

[[nodiscard]] TPerforatorString MakeString(TString str) {
    return new TString{std::move(str)};
}

static TString* UnwrapString(TPerforatorString str) {
    return reinterpret_cast<TString*>(str);
}

extern "C" {

const char* PerforatorStringData(TPerforatorString str) {
    return UnwrapString(str)->data();
}

size_t PerforatorStringSize(TPerforatorString str) {
    return UnwrapString(str)->size();
}

void PerforatorStringDispose(TPerforatorString str) {
    delete UnwrapString(str);
}

} // extern "C"

} // namespace NPerforator::NProfile::NCWrapper
