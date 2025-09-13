#include "aggregating_sampler.hpp"

#include <perforator/proto/pprofprofile/profile.pb.h>

#include <util/generic/string.h>
#include <util/digest/multi.h>

#include <perforator/symbolizer/lib/utils/profile_maps.h>

#include <optional>

namespace NPerforator::NStacksSampling {

namespace {

struct TMappingKey final {
    ui64 MemoryStart{};
    ui64 MemoryLimit{};
    ui64 FileOffset{};
    ui64 Filename{};
    ui64 BuildId{};

    bool operator==(const TMappingKey& other) const = default;

    struct Hash final {
        ui64 operator()(const TMappingKey& key) const noexcept {
            // These 3 should be almost always unique already
            return MultiHash(key.MemoryStart, key.MemoryLimit, key.BuildId);
        }
    };
};

struct TFunctionKey final {
    ui64 Name{};
    ui64 SystemName{};
    ui64 Filename{};
    ui64 StartLine{};

    bool operator==(const TFunctionKey& other) const = default;

    struct Hash final {
        ui64 operator()(const TFunctionKey& key) const noexcept {
            return MultiHash(key.Name, key.SystemName, key.Filename, key.StartLine);
        }
    };
};

struct TLocationKey final {
    ui64 MappingId{};
    ui64 Address{};

    bool operator==(const TLocationKey& other) const = default;

    struct Hash final {
        ui64 operator()(const TLocationKey& key) const noexcept {
            return MultiHash(key.MappingId, key.Address);
        }
    };
};

using TProfileLookup = NUtils::TProfileLookup<NPerforator::NProto::NPProf::Profile>;

class TProfileLocalMappings final {
public:
    std::optional<ui64> GetStringMapping(ui64 id) const {
        return TryFindMapping(StringsMapping_, id);
    }
    ui64 SetStringMapping(ui64 oldId, ui64 newId) {
        StringsMapping_[oldId] = newId;

        return newId;
    }

    //

    std::optional<ui64> GetMappingMapping(const NPerforator::NProto::NPProf::Mapping& mapping) const {
        return TryFindMapping(MappingsMapping_, mapping.id());
    }
    ui64 SetMappingMapping(const NPerforator::NProto::NPProf::Mapping& mapping, ui64 newId) {
        MappingsMapping_[mapping.id()] = newId;

        return newId;
    }

    //

    std::optional<ui64> GetFunctionMapping(const NPerforator::NProto::NPProf::Function& function) const {
        return TryFindMapping(FunctionsMapping_, function.id());
    }
    ui64 SetFunctionMapping(const NPerforator::NProto::NPProf::Function& function, ui64 newId) {
        FunctionsMapping_[function.id()] = newId;

        return newId;
    }

    //

    std::optional<ui64> GetLocationMapping(const NPerforator::NProto::NPProf::Location& location) const {
        return TryFindMapping(LocationsMapping_, location.id());
    }
    ui64 SetLocationMapping(const NPerforator::NProto::NPProf::Location& location, ui64 newId) {
        LocationsMapping_[location.id()] = newId;

        return newId;
    }

private:
    static std::optional<ui64> TryFindMapping(const absl::flat_hash_map<ui64, ui64>& map, ui64 id) {
        const auto it = map.find(id);
        if (it != map.end()) {
            return it->second;
        }

        return std::nullopt;
    }

    absl::flat_hash_map<ui64, ui64> StringsMapping_;
    absl::flat_hash_map<ui64, ui64> MappingsMapping_;
    absl::flat_hash_map<ui64, ui64> FunctionsMapping_;
    absl::flat_hash_map<ui64, ui64> LocationsMapping_;
};

}

class TAggregatingSampler::Impl final {
public:
    Impl() {
        InsertDummyString();
        InsertDummyMapping();
        InsertDummyFunction();
        InsertDummyLocation();
    }

    NPerforator::NProto::NPProf::Profile& GetProfileForMemoryReuse() {
        return ProfileForMemoryReuse_;
    }

    void ValidateAndPopulateSampleType(const TProfileLookup& lookup, TProfileLocalMappings& localMappings) {
        const auto& profile = lookup.GetProfile();

        const auto currentSampleTypeSize = ResultingProfile_.sample_typeSize();
        if (currentSampleTypeSize == 0) {
            for (const auto& valueType : profile.Getsample_type()) {
                NPerforator::NProto::NPProf::ValueType newValueType;
                newValueType.set_type(RemapString(profile, localMappings, valueType.type()));
                newValueType.set_unit(RemapString(profile, localMappings, valueType.unit()));
                ResultingProfile_.mutable_sample_type()->Add(std::move(newValueType));
            }
        } else {
            if (currentSampleTypeSize != profile.sample_typeSize()) {
                throw std::logic_error{"unexpected sample_type size"};
            }

            for (std::size_t i = 0; i < profile.sample_typeSize(); ++i) {
                const auto& currentValueType = ResultingProfile_.sample_type(i);
                const ui64 currentType = currentValueType.type();
                const ui64 currentUnit = currentValueType.unit();

                const auto& newValueType = profile.sample_type(i);
                const ui64 newType = RemapString(profile, localMappings, newValueType.type());
                const ui64 newUnit = RemapString(profile, localMappings, newValueType.unit());

                if (currentType != newType || currentUnit != newUnit) {
                    throw std::logic_error{"sample_type values missmatch"};
                }
            }
        }
    }

    void PopulateComments(const TProfileLookup& lookup, TProfileLocalMappings& localMappings) {
        const auto& profile = lookup.GetProfile();

        for (const auto& comment : profile.Getcomment()) {
            ResultingProfile_.mutable_comment()->Add(RemapString(profile, localMappings, comment));
        }
    }

    void RemapAndAppendSample(
        const TProfileLookup& lookup,
        TProfileLocalMappings& localMappings,
        NPerforator::NProto::NPProf::Sample&& sample) {
        for (auto& label : *sample.mutable_label()) {
            RemapLabel(lookup, localMappings, label);
        }

        for (auto& locationId : *sample.mutable_location_id()) {
            locationId = RemapLocation(lookup, localMappings, lookup.GetLocation(locationId));
        }

        ResultingProfile_.mutable_sample()->Add(std::move(sample));
    }

    const NPerforator::NProto::NPProf::Profile& GetResultingProfile() const noexcept {
        return ResultingProfile_;
    }

private:
    void InsertDummyString() {
        StringsMapping_[""] = ResultingProfile_.string_tableSize();
        ResultingProfile_.mutable_string_table()->Add("");
    }

    void InsertDummyMapping() {
        ResultingProfile_.mutable_mapping()->Add()->set_id(std::numeric_limits<ui64>::max());
    }

    void InsertDummyFunction() {
        ResultingProfile_.mutable_function()->Add()->set_id(std::numeric_limits<ui64>::max());
    }

    void InsertDummyLocation() {
        ResultingProfile_.mutable_location()->Add()->set_id(std::numeric_limits<ui64>::max());
    }

    ui64 RemapLocation(
        const TProfileLookup& lookup,
        TProfileLocalMappings& localMappings,
        const NPerforator::NProto::NPProf::Location& location
    ) {
        if (location.id() == 0) return 0;

        const auto alreadyMappedOpt = localMappings.GetLocationMapping(location);
        if (alreadyMappedOpt.has_value()) {
            return *alreadyMappedOpt;
        }

        const auto mappingId = location.mapping_id() == 0
            ? 0
            : RemapMapping(lookup, localMappings, lookup.GetMapping(location.mapping_id()));

        const TLocationKey locationKey{
            .MappingId = mappingId,
            .Address = location.address(),
        };
        const auto [it, inserted] = LocationsMapping_.try_emplace(locationKey, ResultingProfile_.locationSize());
        const auto locationId = it->second;

        if (inserted) {
            NPerforator::NProto::NPProf::Location newLocation{};
            newLocation.set_id(locationId);
            newLocation.set_mapping_id(locationKey.MappingId);
            newLocation.set_address(locationKey.Address);
            for (const auto& line : location.Getline()) {
                NPerforator::NProto::NPProf::Line newLine{};
                newLine.set_function_id(RemapFunction(lookup, localMappings, lookup.GetFunction(line.function_id())));
                newLine.set_line(line.line());
                newLine.set_column(line.column());
                newLocation.mutable_line()->Add(std::move(newLine));
            }

            ResultingProfile_.mutable_location()->Add(std::move(newLocation));
        }

        return localMappings.SetLocationMapping(location, locationId);
    }

    ui64 RemapMapping(
        const TProfileLookup& lookup,
        TProfileLocalMappings& localMappings,
        const NPerforator::NProto::NPProf::Mapping& mapping
    ) {
        if (mapping.id() == 0) return 0;

        const auto alreadyMappedOpt = localMappings.GetMappingMapping(mapping);
        if (alreadyMappedOpt.has_value()) {
            return *alreadyMappedOpt;
        }

        const auto& profile = lookup.GetProfile();

        const auto filename = RemapString(profile, localMappings, mapping.filename());
        const auto buildId = RemapString(profile, localMappings, mapping.build_id());

        const TMappingKey mappingKey{
            .MemoryStart = mapping.memory_start(),
            .MemoryLimit = mapping.memory_limit(),
            .FileOffset = mapping.file_offset(),
            .Filename = filename,
            .BuildId = buildId,
        };
        const auto [it, inserted] = MappingsMapping_.try_emplace(mappingKey, ResultingProfile_.mappingSize());
        const auto mappingId = it->second;

        if (inserted) {
            NPerforator::NProto::NPProf::Mapping newMapping{};
            newMapping.set_id(mappingId);
            newMapping.set_memory_start(mappingKey.MemoryStart);
            newMapping.set_memory_limit(mappingKey.MemoryLimit);
            newMapping.set_file_offset(mappingKey.FileOffset);
            newMapping.set_filename(mappingKey.Filename);
            newMapping.set_build_id(mappingKey.BuildId);

            ResultingProfile_.mutable_mapping()->Add(std::move(newMapping));
        }

        return localMappings.SetMappingMapping(mapping, mappingId);
    }

    ui64 RemapFunction(
        const TProfileLookup& lookup,
        TProfileLocalMappings& localMappings,
        const NPerforator::NProto::NPProf::Function& function
    ) {
        if (function.id() == 0) return 0;

        const auto alreadyMappedOpt = localMappings.GetFunctionMapping(function);
        if (alreadyMappedOpt.has_value()) {
            return *alreadyMappedOpt;
        }

        const auto& profile = lookup.GetProfile();

        const auto name = RemapString(profile, localMappings, function.name());
        const auto systemName = RemapString(profile, localMappings, function.system_name());
        const auto filename = RemapString(profile, localMappings, function.filename());

        const TFunctionKey functionKey {
            .Name = name,
            .SystemName = systemName,
            .Filename = filename,
            .StartLine = static_cast<ui64>(function.start_line()),
        };
        const auto [it, inserted] = FunctionsMapping_.try_emplace(functionKey, ResultingProfile_.functionSize());
        const auto functionId = it->second;

        if (inserted) {
            NPerforator::NProto::NPProf::Function newFunction{};
            newFunction.set_id(functionId);
            newFunction.set_name(functionKey.Name);
            newFunction.set_system_name(functionKey.SystemName);
            newFunction.set_filename(functionKey.Filename);
            newFunction.set_start_line(functionKey.StartLine);

            ResultingProfile_.mutable_function()->Add(std::move(newFunction));
        }

        return localMappings.SetFunctionMapping(function, functionId);
    }

    ui64 RemapString(
        const NPerforator::NProto::NPProf::Profile& profile,
        TProfileLocalMappings& localMappings,
        ui64 stringIdx
    ) {
        const auto alreadyMappedOpt = localMappings.GetStringMapping(stringIdx);
        if (alreadyMappedOpt.has_value()) {
            return *alreadyMappedOpt;
        }

        const auto [it, inserted] = StringsMapping_.try_emplace(
            profile.string_table(stringIdx),
            ResultingProfile_.string_tableSize()
        );
        const auto newIdx = it->second;

        if (inserted) {
            ResultingProfile_.mutable_string_table()->Add(TString{profile.string_table(stringIdx)});
        }

        return localMappings.SetStringMapping(stringIdx, newIdx);
    }

    void RemapLabel(
        const TProfileLookup& lookup,
        TProfileLocalMappings& localMappings,
        NPerforator::NProto::NPProf::Label& label
    ) {
        const auto& profile = lookup.GetProfile();

        label.set_key(RemapString(profile, localMappings, label.key()));
        label.set_str(RemapString(profile, localMappings, label.str()));
        label.set_num_unit(RemapString(profile, localMappings, label.num_unit()));
    }

    NPerforator::NProto::NPProf::Profile ResultingProfile_;

    absl::flat_hash_map<TString, ui64> StringsMapping_;

    template <typename T>
    using MapType = absl::flat_hash_map<T, ui64, typename T::Hash>;
    MapType<TMappingKey> MappingsMapping_;
    MapType<TFunctionKey> FunctionsMapping_;
    MapType<TLocationKey> LocationsMapping_;

    NPerforator::NProto::NPProf::Profile ProfileForMemoryReuse_;
};

TAggregatingSampler::TAggregatingSampler(ui64 rate)
 : Rate_{rate == 0 ? 1UL : rate},
   Impl_{std::make_unique<Impl>()} {}

TAggregatingSampler::~TAggregatingSampler() = default;

void TAggregatingSampler::AddProfile(TArrayRef<const char> profileBytes) {
    if (profileBytes.data() == nullptr || profileBytes.size() == 0) {
        return;
    }

    auto& profile = Impl_->GetProfileForMemoryReuse();
    if (!profile.ParseFromString(std::string_view{profileBytes.data(), profileBytes.size()})) {
        return;
    }

    AddProfile(profile);
}

void TAggregatingSampler::AddProfile(const NPerforator::NProto::NPProf::Profile& profile) {
    const TProfileLookup lookup{profile};
    TProfileLocalMappings localMappings{};

    Impl_->ValidateAndPopulateSampleType(lookup, localMappings);
    Impl_->PopulateComments(lookup, localMappings);

    for (std::size_t i = 0; i < profile.sampleSize(); ++i) {
        ++SeenStacks_;
        if (SeenStacks_ % Rate_ != 0) {
            // stack is sampled-out
            continue;
        }

        Impl_->RemapAndAppendSample(
            lookup,
            localMappings,
            /* make a copy and consume it */ NPerforator::NProto::NPProf::Sample{profile.sample(i)});
    }
}

const NPerforator::NProto::NPProf::Profile& TAggregatingSampler::GetResultingProfile() const noexcept {
    return Impl_->GetResultingProfile();
}

}
