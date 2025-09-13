#pragma once

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>
#include <library/cpp/containers/absl_flat_hash/flat_hash_set.h>

#include <util/generic/bitmap.h>
#include <util/generic/vector.h>

#include <concepts>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

template <std::unsigned_integral K, std::copy_constructible V, size_t DefaultLittleSize = 1024 * 1024>
class TCompactIntegerMap {
public:
    explicit TCompactIntegerMap(size_t sizeHint = DefaultLittleSize)
        : LittleMapping_(sizeHint)
        , LittleSize_{LittleMapping_.size()}
    {}

    const V& At(K key) const {
        if (IsLittle(key)) {
            return LittleMapping_.at(key).GetRef();
        }
        return BigMapping_.at(key);
    }

    bool TryEmplace(K key, V&& value) {
        if (IsLittle(key)) {
            return TryEmplaceLittle(key, std::move(value));
        } else {
            return TryEmplaceBig(key, std::move(value));
        }
    }

    void EmplaceUnique(K key, V&& value) {
        Y_ENSURE(TryEmplace(key, std::move(value)), "Duplicate key " << key);
    }

    size_t Size() const {
        return Size_;
    }

private:
    bool IsLittle(K key) const {
        return key < LittleSize_;
    }

    bool TryEmplaceLittle(K key, V&& value) {
        auto&& prev = LittleMapping_[key];
        if (prev) {
            return false;
        }
        prev = std::move(value);
        Size_++;
        return true;
    }

    bool TryEmplaceBig(K key, V&& value) {
        auto [_, ok] = BigMapping_.try_emplace(key, std::move(value));
        Size_ += ok;
        return ok;
    }

private:
    size_t Size_ = 0;
    TVector<TMaybe<V>> LittleMapping_;
    absl::flat_hash_map<K, V> BigMapping_;
    size_t LittleSize_ = 0;
};

////////////////////////////////////////////////////////////////////////////////

template <std::unsigned_integral K, size_t DefaultLittleSize = 1024 * 1024>
class TCompactIntegerSet {
public:
    explicit TCompactIntegerSet(size_t sizeHint = DefaultLittleSize)
        : LittleSize_{sizeHint}
    {
        Little_.Reserve(LittleSize_);
    }

    void Insert(K key) {
        if (IsLittle(key)) {
            Little_.Set(key);
        } else {
            Big_.insert(key);
        }
    }

    bool Contains(K key) const {
        if (IsLittle(key)) {
            return Little_.Get(key);
        } else {
            return Big_.contains(key);
        }
    }

private:
    bool IsLittle(K key) const {
        return key < LittleSize_;
    }

private:
    const size_t LittleSize_;
    TDynBitMap Little_;
    absl::flat_hash_set<K> Big_;
};

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
