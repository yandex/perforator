#pragma once

#include "introspection.h"

#include <util/str_stl.h>

/**
 * Hash implementation can change, use persistent hash if you need one
 */

#define Y_GENERATE_T_HASH_AND_EQUALS(NAMESPACE, CPPTYPE)                                                                 \
namespace NAMESPACE {                                                                                                    \
    inline bool operator==(const CPPTYPE& a, const CPPTYPE& b) {                                                         \
        return NIntrospection::Members(a) == NIntrospection::Members(b);                                                 \
    }                                                                                                                    \
}                                                                                                                        \
                                                                                                                         \
template<>                                                                                                               \
struct THash<NAMESPACE::CPPTYPE> {                                                                                       \
    inline size_t operator()(const NAMESPACE::CPPTYPE& item) const noexcept {                                            \
        const auto& hashable = NIntrospection::Members(item);                                                            \
        return THash<std::decay_t<decltype(hashable)>>()(hashable);                                                      \
    }                                                                                                                    \
};
