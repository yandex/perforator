#pragma once

#include <iterator>

#include <util/generic/algorithm.h>
#include <util/generic/array_ref.h>
#include <util/generic/vector.h>


template <typename It, typename T = typename std::iterator_traits<It>::value_type>
TVector<T> ApplyPermutation(It begin, It end, TConstArrayRef<size_t> permutation) {
    static_assert(std::is_base_of_v<std::random_access_iterator_tag, typename std::iterator_traits<It>::iterator_category>);
    using TIteratorDifference = typename std::iterator_traits<It>::difference_type;

    Y_ENSURE(static_cast<TIteratorDifference>(std::distance(begin, end)) == static_cast<TIteratorDifference>(permutation.size()));

    TVector<T> result;
    result.reserve(std::distance(begin, end));
    for (size_t i : permutation) {
        result.push_back(*(begin + i));
    }
    return result;
}

template <class T>
TVector<T> ApplyPermutation(TConstArrayRef<T> origin, TConstArrayRef<size_t> permutation) {
    return ApplyPermutation(origin.begin(), origin.end(), permutation);
}

template <class C>
auto ApplyPermutation(C&& container, TConstArrayRef<size_t> permutation) {
    using std::begin;
    using std::end;
    return ApplyPermutation(begin(container), end(container), permutation);
}

template <class C>
void ApplyPermutationInplace(C&& container, TConstArrayRef<size_t> permutation) {
    auto res = ApplyPermutation(container, permutation);
    for (auto i = 0; i < std::size(container); ++i) {
        container[i] = std::move(res[i]);
    }
}

template <typename C>
TVector<size_t> MakeSortedPermutation(C&& values) {
    TVector<size_t> idx(values.size(), 0);
    Iota(idx.begin(), idx.end(), 0);
    StableSortBy(idx, [&values](size_t i) -> const auto& {
        return values[i];
    });
    return idx;
}

template <typename Base, typename ...Ts>
void MultiSort(Base& base, Ts& ...arrays) {
    auto permutation = MakeSortedPermutation(base);
    ApplyPermutationInplace(base, permutation);
    [[maybe_unused]] auto _ = ((ApplyPermutationInplace(arrays, permutation), 0) + ... + 0);
}
