#pragma once

#include <perforator/lib/profile/entity_index.h>
#include <perforator/lib/profile/profile.h>
#include <perforator/lib/profile/trie/trie.h>
#include <perforator/proto/profile/render_options.pb.h>

namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

// Frame identity for deduplication
struct TFlameFrameKey {
    TStringId NameId = TStringId::Invalid();
    TStringId FileId = TStringId::Invalid();
    TBinaryId BinaryId = TBinaryId::Zero();
    ui32 Line = 0;
    ui32 Column = 0;

    bool operator==(const TFlameFrameKey& other) const = default;

    template <typename H>
    friend H AbslHashValue(H h, const TFlameFrameKey& key) {
        return H::combine(std::move(h), key.NameId, key.FileId, key.BinaryId, key.Line, key.Column);
    }

    // Special marker for truncated stack
    static TFlameFrameKey TruncatedStack() {
        return TFlameFrameKey{
            .NameId = TStringId::Invalid(),
            .FileId = TStringId::Invalid(),
            .BinaryId = TBinaryId::Zero(),
            .Line = std::numeric_limits<ui32>::max(),
            .Column = std::numeric_limits<ui32>::max(),
        };
    }

    bool IsTruncatedStack() const {
        return Line == std::numeric_limits<ui32>::max() &&
            Column == std::numeric_limits<ui32>::max();
    }
};

// Flamegraph node identity - root, label, or frame
using TFlameNodeId = std::variant<std::monostate, TLabelId, TFlameFrameKey>;

struct TFlameValue {
    i64 SampleCount = 0;
    i64 EventCount = 0;

    TFlameValue& operator+=(const TFlameValue& other) {
        SampleCount += other.SampleCount;
        EventCount += other.EventCount;
        return *this;
    }
};

// The trie is keyed by a dense ui32 node-id (interned TFlameNodeId), not the
// variant itself. This shrinks the edge-map key to {ui32 key, ui32 parent} = 8 bytes
// (vs ~28 for the variant), so the probe table is ~3x smaller and far more cache-
// resident on the memory-latency-bound descend. denseToNode maps id -> identity for
// rendering.
using TFlameTrie = TTrie<ui32, TFlameValue>;

// A built flame trie together with the table mapping each dense node id back to its
// identity (root / label / frame). Rendering needs the table to resolve names; the two
// are produced together by BuildFlameTrie and consumed together by RenderTrieToJson, so
// they travel as one value rather than a trie plus an out-parameter.
struct TFlameTrieResult {
    TFlameTrie Trie;
    TVector<TFlameNodeId> DenseToNode;
};

TFlameTrieResult BuildFlameTrie(
    TProfile profile,
    const NProto::NProfile::RenderOptions& options,
    TValueTypeId sampleTypeIndex
);

TValueTypeId GuessDefaultSampleType(TProfile profile);

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
