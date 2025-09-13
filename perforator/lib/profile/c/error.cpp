#include "error.hpp"

#include <util/generic/yexception.h>


namespace NPerforator::NProfile::NCWrapper {

extern "C" {

const char* PerforatorErrorString(TPerforatorError err) {
    if (!err) {
        return nullptr;
    }
    return UnwrapError(err)->Message().data();
}

void PerforatorErrorDispose(TPerforatorError err) {
    TError::Dispose(UnwrapError(err));
}

} // extern "C"

} // namespace NPerforator::NProfile::NCWrapper
