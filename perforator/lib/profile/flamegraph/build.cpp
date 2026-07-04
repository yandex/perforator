#include "build.h"

#include <perforator/lib/profile/profile.h>
#include <perforator/lib/profile/trie/trie.h>
#include <perforator/lib/profile/well_known.h>

#include <library/cpp/containers/absl/flat_hash_map.h>

#include <contrib/libs/rapidjson/include/rapidjson/writer.h>
#include <contrib/libs/rapidjson/include/rapidjson/stringbuffer.h>

#include <util/generic/hash.h>
#include <util/generic/yexception.h>
#include <util/memory/pool.h>
#include <util/string/cast.h>

#include <limits>
#include <type_traits>

namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

// Resolve sample type index from default_sample_type metadata.
// Uses pprof behavior: match DefaultSampleType by Type string, or fall back to last sample type.
TValueTypeId GuessDefaultSampleType(TProfile profile) {
    Y_ENSURE(profile.ValueTypes().size() > 0, "profile has no sample types");

    const auto& metadata = profile.GetMetadata();

    // If default_sample_type is set, find matching value type by string comparison (like pprof does)
    if (metadata.default_sample_type() > 0) {
        TStringBuf defaultType = profile.String(TStringId::FromInternalIndex(metadata.default_sample_type())).View();
        for (TValueType valueType : profile.ValueTypes()) {
            if (valueType.GetType().View() == defaultType) {
                return valueType.GetIndex();
            }
        }
    }

    // Fall back to last sample type (pprof default behavior)
    return profile.ValueTypes().back().GetIndex();
}

////////////////////////////////////////////////////////////////////////////////

namespace {

////////////////////////////////////////////////////////////////////////////////

// Outcome of descending one root-side stack segment, stored in the segment memo.
// A trie node sits at a unique depth, so (startNode, segmentId) pins the entry depth — the
// same pair yields identical frames and identical cap behavior every time, making the
// outcome deterministic and replayable: Length advances depth on replay, Truncated stops
// the stack (emitting the "(truncated stack)" marker). Bitfields keep it ui64-sized so it
// fits the memo value slot without packing.
struct TSegmentOutcome {
    ui32 EndNode = 0;
    ui32 Length : 31 = 0;
    ui32 Truncated : 1 = 0;
};
// Stored by value in the memo: must be a trivially-copyable POD that fits the ui64 slot.
static_assert(sizeof(TSegmentOutcome) == sizeof(ui64));
static_assert(std::is_standard_layout_v<TSegmentOutcome>);
static_assert(std::is_trivially_copyable_v<TSegmentOutcome>);

// Segment-memo key: the (startNode id, segment id) pair the outcome is cached against.
ui64 SegmentMemoKey(ui32 startNodeId, ui32 segmentId) {
    return (static_cast<ui64>(startNodeId) << 32) | segmentId;
}

} // anonymous namespace

TFlameTrieResult BuildFlameTrie(
    TProfile profile,
    const NProto::NProfile::RenderOptions& options,
    TValueTypeId sampleTypeIndex
) {
    auto keyIds = TWellKnownLabelKeyIds::Build(profile);

    TFlameTrieResult result;
    TFlameTrie& trie = result.Trie;
    TVector<TFlameNodeId>& denseToNode = result.DenseToNode;

    // Intern each distinct node identity to a dense ui32 (id 0 == root). The trie
    // is then keyed by ui32 instead of the variant, shrinking the edge-map key from ~28 to
    // 8 bytes (far more cache-resident). denseToNode maps id -> identity for rendering.
    absl::flat_hash_map<TFlameNodeId, ui32> nodeIntern;
    denseToNode.clear();
    auto intern = [&](const TFlameNodeId& id) -> ui32 {
        auto [it, inserted] = nodeIntern.try_emplace(id, static_cast<ui32>(denseToNode.size()));
        if (inserted) {
            denseToNode.push_back(id);
        }
        return it->second;
    };
    intern(TFlameNodeId{});  // root identity -> dense 0

    TVector<TFlameValue> keyValues(profile.SampleKeys().size());
    for (auto sample : profile.Samples()) {
        ui32 keyIndex = sample.GetKey().GetIndex().GetInternalIndex();
        keyValues[keyIndex].SampleCount += 1;
        keyValues[keyIndex].EventCount += sample.GetValues()[*sampleTypeIndex].GetValue();
    }

    // Precompute per-stack-frame flamegraph node keys (as dense ids) once, instead
    // of rebuilding TFrameKey from profile accessors on every descend. frameFlat[k] for k
    // in [frameOff[fi], frameOff[fi + 1]) are the dense ids of frame fi's inline chain
    // (root-to-leaf within the frame).
    const bool showFiles = options.show_file_names();
    const bool showLines = options.show_line_numbers();
    const ui32 frameCount = profile.StackFrames().size();
    TVector<ui32> frameOff(frameCount + 1, 0);
    TVector<ui32> frameFlat;
    for (TStackFrame frame : profile.StackFrames()) {
        ui32 fi = frame.GetIndex().GetInternalIndex();
        frameOff[fi] = frameFlat.size();
        TBinaryId binaryId = frame.GetBinary().GetIndex();
        TStringId binaryPathId = binaryId != TBinaryId::Zero()
            ? frame.GetBinary().GetPath().GetIndex()
            : TStringId::Invalid();
        TInlineChain chain = frame.GetInlineChain();
        if (chain.GetLines().empty()) {
            frameFlat.push_back(intern(TFlameNodeId{TFlameFrameKey{
                .NameId = TStringId::Invalid(),
                .FileId = showFiles ? binaryPathId : TStringId::Invalid(),
                .BinaryId = binaryId,
                .Line = 0,
                .Column = 0,
            }}));
        } else {
            // Inline chains are stored innermost-first (leaf-to-root); flamegraph needs
            // root-to-leaf, so iterate in reverse (outermost caller descends first).
            for (TSourceLine line : chain.GetLines() | std::views::reverse) {
                TFunction func = line.GetFunction();
                TStringId fileId = func.GetFileName().GetIndex();
                frameFlat.push_back(intern(TFlameNodeId{TFlameFrameKey{
                    .NameId = func.GetName().GetIndex(),
                    .FileId = showFiles ? fileId : TStringId::Invalid(),
                    .BinaryId = binaryId,
                    .Line = showLines ? line.GetLine() : 0,
                    .Column = showLines ? line.GetColumn() : 0,
                }}));
            }
        }
    }
    frameOff[frameCount] = frameFlat.size();

    // Effective depth cap; max_depth == 0 means unlimited.
    const ui32 depthCap = options.max_depth() == 0
        ? std::numeric_limits<ui32>::max()
        : options.max_depth();
    const ui32 truncatedDense = intern(TFlameNodeId{TFlameFrameKey::TruncatedStack()});

    // Memoize shared root-side segments: SegmentMemoKey -> TSegmentOutcome. Many samples
    // share a root-side path, so this replays the segment instead of re-descending it. With
    // no cap nothing truncates, so every entry is a full segment.
    absl::flat_hash_map<ui64, TSegmentOutcome> segMemo;

    for (auto sampleKey : profile.SampleKeys()) {
        ui32 keyIndex = sampleKey.GetIndex().GetInternalIndex();
        TFlameValue value = keyValues[keyIndex];
        if (value.SampleCount == 0) {
            continue;
        }

        auto node = trie.Root();
        ui32 depth = 0;  // labels + frames descended; drives depth-limit truncation

        // descend builds structure only; the value is added once at the leaf below
        // and propagated up by ReduceToRoot after the whole build.
        auto descend = [&](ui32 denseId) {
            node = node.GetOrCreateChild(denseId);
            ++depth;
        };

        // Label order: containers, pid, process name, thread name, signal. ThreadId is
        // intentionally excluded (only ThreadCommand is rendered).
        bool hasFirstContainer = false;
        std::array<TLabelId, 4> labelNodes{{
            TLabelId::Invalid(),  // ProcessId (pid)
            TLabelId::Invalid(),  // ProcessCommand (process name)
            TLabelId::Invalid(),  // ThreadCommand (thread name)
            TLabelId::Invalid(),  // SignalName
        }};
        for (auto label : sampleKey.GetLabels()) {
            auto labelType = keyIds.GetLabel(label.GetKey().GetIndex());
            if (!labelType) {
                continue;
            }
            switch (*labelType) {
                case NProto::NProfile::Workload:
                    if (label.IsString()) {
                        if (!hasFirstContainer && label.GetString().View().StartsWith("iss_hook_")) {
                            hasFirstContainer = true;
                            break;
                        }
                        hasFirstContainer = true;
                        descend(intern(TFlameNodeId{label.GetIndex()}));
                    }
                    break;
                case NProto::NProfile::ProcessId:
                    if (label.IsNumber()) {
                        labelNodes[0] = label.GetIndex();
                    }
                    break;
                case NProto::NProfile::ProcessCommand:
                    if (label.IsString() && label.GetString().GetIndex() != TStringId::Zero()) {
                        labelNodes[1] = label.GetIndex();
                    }
                    break;
                case NProto::NProfile::ThreadId:
                    break;  // intentionally skipped
                case NProto::NProfile::ThreadCommand:
                    if (label.IsString() && label.GetString().GetIndex() != TStringId::Zero()) {
                        labelNodes[2] = label.GetIndex();
                    }
                    break;
                case NProto::NProfile::SignalName:
                    if (label.IsString() && label.GetString().GetIndex() != TStringId::Zero()) {
                        labelNodes[3] = label.GetIndex();
                    }
                    break;
                default:
                    break;
            }
        }
        for (TLabelId labelId : labelNodes) {
            if (labelId.IsValid()) {
                descend(intern(TFlameNodeId{labelId}));
            }
        }

        // Per stack: replay the shared root-side segment from the memo, or descend it once
        // and record the outcome. The outcome is deterministic per (startNode, segmentId)
        // — see segMemo above — so a truncation is cached exactly like a full descent. Frame
        // order is reverse(GetFrames) = reverse(segment) + TopFrame. On truncation we stop
        // and emit one "(truncated stack)" node. With no cap the truncated bit is never set,
        // so this is the plain segment-memo fast path.
        bool truncated = false;
        for (TStack stack : sampleKey.GetStacks() | std::views::reverse) {
            if (truncated) {
                break;
            }
            TStackSegment segment = stack.GetStackSegment();
            ui64 memoKey = SegmentMemoKey(node.GetId(), segment.GetIndex().GetInternalIndex());

            auto [it, inserted] = segMemo.try_emplace(memoKey);
            if (!inserted) {
                // Deterministic replay: jump to the recorded end, advancing depth so later
                // segments see the right cap, and stop here if this segment was truncated.
                node = trie.NodeAt(it->second.EndNode);
                depth += it->second.Length;
                if (it->second.Truncated) {
                    truncated = true;
                    break;
                }
            } else {
                // First time for this (startNode, segmentId): descend frame-by-frame,
                // truncating on the cap, then record the outcome (full or partial). The
                // iterator stays valid across the descent — it touches the trie, not segMemo.
                const ui32 startDepth = depth;
                for (TStackFrame frame : segment.GetFrames() | std::views::reverse) {
                    ui32 fi = frame.GetIndex().GetInternalIndex();
                    for (ui32 k = frameOff[fi]; k < frameOff[fi + 1]; ++k) {
                        if (depth >= depthCap) {
                            truncated = true;
                            break;
                        }
                        node = node.GetOrCreateChild(frameFlat[k]);
                        ++depth;
                    }
                    if (truncated) {
                        break;
                    }
                }
                it->second = TSegmentOutcome{
                    .EndNode = node.GetId(),
                    .Length = depth - startDepth,
                    .Truncated = truncated,
                };
                if (truncated) {
                    break;
                }
            }

            TStackFrame top = stack.GetTopFrame();
            ui32 tfi = top.GetIndex().GetInternalIndex();
            for (ui32 k = frameOff[tfi]; k < frameOff[tfi + 1]; ++k) {
                if (depth >= depthCap) {
                    truncated = true;
                    break;
                }
                node = node.GetOrCreateChild(frameFlat[k]);
                ++depth;
            }
        }
        if (truncated) {
            descend(truncatedDense);
        }

        node.GetValue() += value;  // self-count at the leaf
    }

    trie.ReduceToRoot([](TFlameValue& parent, const TFlameValue& child) {
        parent += child;
    });
    trie.Finalize();
    return result;
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
