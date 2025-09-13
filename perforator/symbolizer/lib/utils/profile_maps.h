#pragma once

#include <util/system/types.h>

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>

#include <perforator/proto/pprofprofile/profile.pb.h>

#include <format>
#include <vector>
#include <stdexcept>

namespace NPerforator::NUtils {

template <typename Traits>
class TItemByIdMap final {
public:
    using value_type = typename Traits::value_type;

    explicit TItemByIdMap(const typename Traits::ProfileType& profile) {
        SmallIdMap_.assign(Traits::Size(profile) + 1, nullptr);

        for (std::size_t i = 0; i < Traits::Size(profile); ++i) {
            const auto& item = Traits::At(profile, i);
            const auto itemId = item.id();
            if (itemId < SmallIdMap_.size()) {
                SmallIdMap_[itemId] = &item;
            } else {
                BigIdMap_.emplace(itemId, &item);
            }
        }
    }

    const value_type& At(ui64 itemId) const {
        const value_type* itemPtr = nullptr;

        if (itemId < SmallIdMap_.size()) {
            itemPtr = SmallIdMap_[itemId];
        } else {
            itemPtr = BigIdMap_.at(itemId);
        }

        if (itemPtr == nullptr) {
            throw std::logic_error{std::format("No item with id {}", itemId)};
        }

        return *itemPtr;
    }

private:
    std::vector<const value_type*> SmallIdMap_;
    absl::flat_hash_map<ui64, const value_type*> BigIdMap_;
};

template <typename TProtoProfile>
struct LocationByIdTraits final {
    using value_type = NPerforator::NProto::NPProf::Location;
    using ProfileType = TProtoProfile;

    static std::size_t Size(const ProfileType& profile) {
        return profile.locationSize();
    }

    static const value_type& At(const ProfileType& profile, std::size_t i) {
        return profile.location(i);
    }
};

template <typename TProtoProfile>
struct MappingByIdTraits final {
    using value_type = NPerforator::NProto::NPProf::Mapping;
    using ProfileType = TProtoProfile;

    static std::size_t Size(const ProfileType& profile) {
        return profile.mappingSize();
    }

    static const value_type& At(const ProfileType& profile, std::size_t i) {
        return profile.mapping(i);
    }
};

template <typename TProtoProfile>
struct FunctionByIdTraits final {
    using value_type = NPerforator::NProto::NPProf::Function;
    using ProfileType = TProtoProfile;

    static std::size_t Size(const ProfileType& profile) {
        return profile.functionSize();
    }

    static const value_type& At(const ProfileType& profile, std::size_t i) {
        return profile.function(i);
    }
};

template <typename TProtoProfile>
class TProfileLookup final {
public:
    explicit TProfileLookup(const TProtoProfile& profile)
    : Profile_{profile},
      LocationByIdMap_{Profile_},
      MappingByIdMap_{Profile_},
      FunctionByIdMap_{Profile_} {}

    const TProtoProfile& GetProfile() const noexcept {
        return Profile_;
    }

    const NPerforator::NProto::NPProf::Location& GetLocation(ui64 id) const {
        return LocationByIdMap_.At(id);
    }

    const NPerforator::NProto::NPProf::Mapping& GetMapping(ui64 id) const {
        return MappingByIdMap_.At(id);
    }

    const NPerforator::NProto::NPProf::Function& GetFunction(ui64 id) const {
        return FunctionByIdMap_.At(id);
    }

private:
    const TProtoProfile& Profile_;

    TItemByIdMap<LocationByIdTraits<TProtoProfile>> LocationByIdMap_;
    TItemByIdMap<MappingByIdTraits<TProtoProfile>> MappingByIdMap_;
    TItemByIdMap<FunctionByIdTraits<TProtoProfile>> FunctionByIdMap_;
};

}
