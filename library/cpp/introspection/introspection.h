#pragma once

#include <pfr/core.hpp>
#include <pfr/tuple_size.hpp>

namespace NIntrospection {
    // @brief returns count of members of structure T
    // @complexity templates instantiation count is log(sizeof(T))
    // @note if structure contains static array, each element in array will be count as member
    template <class T>
    constexpr size_t MembersCount() {
        return pfr::tuple_size_v<T>;
    }

    // @brief get references to members of instance of T as tuple.
    // @note supports both const and non-const introspection.
    template <class T>
    constexpr decltype(auto) Members(T& v) {
        return pfr::structure_tie(v);
    }

    // @brief get reference or const reference to I-th member of instance of T
    template <size_t I, class T>
    constexpr decltype(auto) Member(T&& v) {
        return pfr::get<I>(std::forward<T>(v));
    }
}
