#pragma once

#include "entity_index.h"

#include <perforator/proto/profile/well_known_labels.pb.h>

#include <library/cpp/containers/absl/flat_hash_map.h>

#include <optional>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

class TProfile;

class TWellKnownLabelKeyIds {
public:
    static TWellKnownLabelKeyIds Build(const TProfile& profile);

    std::optional<NProto::NProfile::WellKnownLabel> GetLabel(TStringId keyId) const;

private:
    absl::flat_hash_map<TStringId, NProto::NProfile::WellKnownLabel> KeyToLabel_;
};

// Canonical key for a well-known label.
TStringBuf GetWellKnownLabelKey(NProto::NProfile::WellKnownLabel kind);

// Canonical + deprecated keys for a well-known label.
TConstArrayRef<TString> GetAllWellKnownLabelKeys(NProto::NProfile::WellKnownLabel kind);

// All well-known labels that have at least one key defined.
TConstArrayRef<NProto::NProfile::WellKnownLabel> GetWellKnownLabels();

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
