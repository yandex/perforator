#pragma once

#include "entity_index.h"

#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/containers/absl/flat_hash_map.h>

namespace NPerforator::NProfile {

class TStringTableSubview {
public:
    explicit TStringTableSubview(
        const NProto::NProfile::StringTable* strtab,
        TStringId index
    )
        : StringTable_{strtab}
        , Index_{index}
    {}

    TStringId GetIndex() const {
        return Index_;
    }

    const char* Data() const {
        return StringTable_->strings().data() + Offset();
    }

    ui32 Offset() const {
        return StringTable_->offset(*Index_);
    }

    ui32 Size() const {
        return StringTable_->length(*Index_);
    }

    const char* begin() const {
        return Data();
    }

    const char* end() const {
        return Data() + Size();
    }

    // The returned TStringBuf can be invalidated on string table modification,
    // so it should not be stored for a long time.
    TStringBuf AsStringBuf() const {
        return {Data(), Size()};
    }

    std::string_view AsStringView() const {
        return AsStringBuf();
    }

    bool operator==(const TStringTableSubview& other) const noexcept {
        return (StringTable_ == other.StringTable_ && Index_ == other.Index_)
            || AsStringBuf() == other.AsStringBuf();
    }

    bool operator!=(const TStringTableSubview& other) const noexcept {
        return !operator==(other);
    }

    // NOLINTNEXTLINE(readability-identifier-naming)
    struct absl_container_eq {
        using is_transparent = void;

        bool operator()(TStringTableSubview lhs, TStringTableSubview rhs) const {
            return lhs == rhs;
        }

        bool operator()(std::string_view lhs, TStringTableSubview rhs) const {
            return operator()(lhs, rhs.AsStringView());
        }

        bool operator()(TStringTableSubview lhs, std::string_view rhs) const {
            return operator()(lhs.AsStringView(), rhs);
        }

        bool operator()(std::string_view lhs, std::string_view rhs) const {
            return lhs == rhs;
        }
    };

    // NOLINTNEXTLINE(readability-identifier-naming)
    struct absl_container_hash {
        using is_transparent = void;

        size_t operator()(TStringTableSubview value) const {
            return operator()(value.AsStringView());
        }

        size_t operator()(std::string_view value) const {
            return absl::HashOf(value);
        }
    };

private:
    const NProto::NProfile::StringTable* StringTable_;
    TStringId Index_;
};

class TStringInterner {
public:
    explicit TStringInterner(NProto::NProfile::StringTable* table)
        : StringTable_(table)
    {
        Y_ENSURE(table);
        Y_ENSURE(table->offset_size() == table->length_size());
        for (i32 i = 0; i < table->offset_size(); ++i) {
            const size_t offset = table->offset(i);
            Y_ENSURE(offset <= table->strings().size());
            Y_ENSURE(table->length(i) <= table->strings().size() - offset);
            auto id = TStringId::FromInternalIndex(i);
            Strings_.emplace(Get(id), id);
        }
    }

    TStringTableSubview Intern(TStringBuf value) {
        if (auto it = Strings_.find(value); it != Strings_.end()) {
            return it->first;
        }
        auto id = TStringId::FromInternalIndex(StringTable_->offset_size());
        StringTable_->add_offset(StringTable_->strings().size());
        StringTable_->add_length(value.size());
        StringTable_->mutable_strings()->append(value);
        auto view = Get(id);
        Strings_.emplace(view, id);
        return view;
    }

    TStringTableSubview Get(TStringId id) const {
        return TStringTableSubview{StringTable_, id};
    }

private:
    NProto::NProfile::StringTable* StringTable_;
    absl::flat_hash_map<TStringTableSubview, TStringId> Strings_;
};

}
