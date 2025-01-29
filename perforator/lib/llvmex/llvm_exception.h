#pragma once

#include <util/generic/yexception.h>


class TLLVMException : public yexception {};

#define Y_LLVM_RAISE(...) \
    [&](auto&& expected) { \
        if (auto err = expected.takeError()) { \
            throw TLLVMException{} << toString(std::move(err)); \
        } \
        return std::move(*expected); \
    }(__VA_ARGS__)

#define Y_LLVM_UNWRAP(var, expected, ...) \
    if (!expected) { \
        auto error = expected.takeError(); \
        __VA_ARGS__ \
    } \
    auto var = *expected;\
