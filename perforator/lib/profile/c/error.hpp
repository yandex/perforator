#pragma once

#include "error.h"

#include <util/generic/string.h>
#include <util/generic/yexception.h>

#include <concepts>


namespace NPerforator::NProfile::NCWrapper {

class TError {
public:
    TStringBuf Message() const {
        return Message_;
    }

    static TError* Capture() {
        return new TError{CurrentExceptionMessage()};
    }

    static void Dispose(TError* error) {
        delete error;
    }

private:
    TError(TString message)
        : Message_{std::move(message)}
    {}

private:
    TString Message_;
};

inline TError* UnwrapError(TPerforatorError error) {
    return reinterpret_cast<TError*>(error);
}

template <std::invocable<> F>
requires std::is_void_v<std::invoke_result_t<F>>
TPerforatorError InterceptExceptions(F&& func) {
    try {
        func();
        return nullptr;
    } catch (...) {
        return TError::Capture();
    }
}

} // namespace NPerforator::NProfile::NCWrapper
