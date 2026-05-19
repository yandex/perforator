#include "profile.h"

#include <util/generic/string.h>
#include <util/generic/vector.h>
#include <util/stream/output.h>

#include <array>

namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

namespace {

// Index 0 is the main key, rest are deprecated keys.
// We store TString to own the data, since proto descriptor strings
// come from a dynamically allocated descriptor pool.
const std::array<TVector<TString>, NProto::NProfile::WellKnownLabel_ARRAYSIZE>& GetWellKnownLabelKeysMap() {
    static const auto map = [] {
        std::array<TVector<TString>, NProto::NProfile::WellKnownLabel_ARRAYSIZE> result;
        const auto* descriptor = NProto::NProfile::WellKnownLabel_descriptor();
        for (int i = 0; i < descriptor->value_count(); ++i) {
            const auto* valueDesc = descriptor->value(i);
            int index = valueDesc->number();
            if (index < 0 || static_cast<size_t>(index) >= result.size()) {
                continue;
            }
            const auto& opts = valueDesc->options();
            if (opts.HasExtension(NProto::NProfile::label_key)) {
                result[index].push_back(TString{opts.GetExtension(NProto::NProfile::label_key)});
            }
            for (const auto& key : opts.GetRepeatedExtension(NProto::NProfile::deprecated_label_key)) {
                result[index].push_back(TString{key});
            }
        }
        return result;
    }();
    return map;
}

} // namespace

////////////////////////////////////////////////////////////////////////////////

TStringBuf GetWellKnownLabelKey(NProto::NProfile::WellKnownLabel kind) {
    const auto& keys = GetWellKnownLabelKeysMap()[kind];
    return keys.empty() ? TStringBuf{} : keys[0];
}

TConstArrayRef<TString> GetAllWellKnownLabelKeys(NProto::NProfile::WellKnownLabel kind) {
    return GetWellKnownLabelKeysMap()[kind];
}

TConstArrayRef<NProto::NProfile::WellKnownLabel> GetWellKnownLabels() {
    static const auto labels = [] {
        TVector<NProto::NProfile::WellKnownLabel> result;
        const auto* descriptor = NProto::NProfile::WellKnownLabel_descriptor();
        const auto& map = GetWellKnownLabelKeysMap();
        for (int i = 0; i < descriptor->value_count(); ++i) {
            int index = descriptor->value(i)->number();
            Y_ENSURE(
                index >= 0 && static_cast<size_t>(index) < map.size() && !map[index].empty(),
                "WellKnownLabel " << descriptor->value(i)->name() << " has no label_key defined");
            result.push_back(static_cast<NProto::NProfile::WellKnownLabel>(index));
        }
        return result;
    }();
    return labels;
}

////////////////////////////////////////////////////////////////////////////////

TProfile::TProfile(const NProto::NProfile::Profile* profile)
    : Profile_{profile}
{}

const NProto::NProfile::Metadata& TProfile::GetMetadata() const {
    return Profile_->metadata();
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile

////////////////////////////////////////////////////////////////////////////////

template <>
void Out<NPerforator::NProfile::TProfileString>(
    IOutputStream& stream,
    const NPerforator::NProfile::TProfileString& ref
) {
    Out<TStringBuf>(stream, ref.View());
}

////////////////////////////////////////////////////////////////////////////////
