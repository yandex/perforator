#include "render.h"

#include <perforator/lib/profile/merge.h>
#include <perforator/lib/profile/profile.h>
#include <perforator/lib/profile/trie/trie.h>

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>

#include <contrib/libs/rapidjson/include/rapidjson/writer.h>
#include <contrib/libs/rapidjson/include/rapidjson/stringbuffer.h>

#include <library/cpp/iterator/enumerate.h>

#include <util/generic/hash.h>
#include <util/memory/pool.h>
#include <util/string/cast.h>

#include <algorithm>
#include <limits>
#include <type_traits>

namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

namespace {

// Frame identity for deduplication
struct TFrameKey {
    TStringId NameId = TStringId::Invalid();
    TStringId FileId = TStringId::Invalid();
    TBinaryId BinaryId = TBinaryId::Zero();
    ui32 Line = 0;
    ui32 Column = 0;

    bool operator==(const TFrameKey& other) const = default;

    template <typename H>
    friend H AbslHashValue(H h, const TFrameKey& key) {
        return H::combine(std::move(h), key.NameId, key.FileId, key.BinaryId, key.Line, key.Column);
    }

    // Special marker for truncated stack
    static TFrameKey TruncatedStack() {
        return TFrameKey{
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
using TFlameNodeId = std::variant<std::monostate, TLabelId, TFrameKey>;

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

////////////////////////////////////////////////////////////////////////////////

class TLabelKeyIds {
public:
    static TLabelKeyIds Build(TProfile& profile) {
        TLabelKeyIds ids;

        THashMap<TStringBuf, NProto::NProfile::WellKnownLabel> keyToLabel;
        for (auto label : GetWellKnownLabels()) {
            for (const TString& key : GetAllWellKnownLabelKeys(label)) {
                keyToLabel.emplace(key, label);
            }
        }

        for (TProfileString str : profile.Strings()) {
            if (auto it = keyToLabel.find(str.View()); it != keyToLabel.end()) {
                ids.KeyToLabel_.emplace(str.GetIndex(), it->second);
            }
        }

        return ids;
    }

    std::optional<NProto::NProfile::WellKnownLabel> GetLabel(TStringId keyId) const {
        auto it = KeyToLabel_.find(keyId);
        return it != KeyToLabel_.end() ? std::optional{it->second} : std::nullopt;
    }

private:
    absl::flat_hash_map<TStringId, NProto::NProfile::WellKnownLabel> KeyToLabel_;
};

////////////////////////////////////////////////////////////////////////////////

bool IsInvalidFunctionName(TStringBuf name) {
    return name.empty() || name == "??" || name == "<invalid>";
}

bool IsInvalidFilename(TStringBuf name) {
    return name.empty() || name == "??" || name == "<invalid>" || name == "<unknown>";
}

TStringBuf SanitizeFileName(TStringBuf name) {
    name.SkipPrefix("/-B") || name.SkipPrefix("/-S");
    return name;
}

////////////////////////////////////////////////////////////////////////////////

// Resolve sample type index from default_sample_type metadata.
// Uses pprof behavior: match DefaultSampleType by Type string, or fall back to last sample type.
ui32 ResolveSampleTypeIndex(TProfile& profile) {
    Y_ENSURE(profile.ValueTypes().size() > 0, "profile has no sample types");

    const auto& metadata = profile.GetMetadata();

    // If default_sample_type is set, find matching value type by string comparison (like pprof does)
    if (metadata.default_sample_type() > 0) {
        TStringBuf defaultType = profile.String(TStringId::FromInternalIndex(metadata.default_sample_type())).View();
        for (auto [index, valueType] : Enumerate(profile.ValueTypes())) {
            if (valueType.GetType().View() == defaultType) {
                return index;
            }
        }
    }

    // Fall back to last sample type (pprof default behavior)
    return profile.ValueTypes().size() - 1;
}

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

TFlameTrieResult BuildFlameTrie(
    TProfile& profile,
    const TLabelKeyIds& keyIds,
    const NProto::NProfile::RenderOptions& options,
    ui32 sampleTypeIndex
) {
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
        keyValues[keyIndex].EventCount += sample.GetValues()[sampleTypeIndex].GetValue();
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
            frameFlat.push_back(intern(TFlameNodeId{TFrameKey{
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
                if (IsInvalidFilename(func.GetFileName().View())) {
                    fileId = binaryPathId;
                }
                frameFlat.push_back(intern(TFlameNodeId{TFrameKey{
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
    const ui32 truncatedDense = intern(TFlameNodeId{TFrameKey::TruncatedStack()});

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

// String table for JSON output - pool provides stable memory-local storage
class TStringInterner {
public:
    TStringInterner()
        : Pool_(4096)
    {}

    ui32 Intern(TStringBuf s) {
        if (auto it = StringToId_.find(s); it != StringToId_.end()) {
            return it->second;
        }
        ui32 id = Strings_.size();
        TStringBuf interned = Pool_.AppendString(s);
        Strings_.push_back(interned);
        StringToId_.emplace(interned, id);
        return id;
    }

    // Intern a string that's guaranteed to outlive this interner (e.g., from profile)
    // Avoids copying to pool
    ui32 InternStable(TStringBuf s) {
        if (auto it = StringToId_.find(s); it != StringToId_.end()) {
            return it->second;
        }
        ui32 id = Strings_.size();
        Strings_.push_back(s);  // No copy - string is stable
        StringToId_.emplace(s, id);
        return id;
    }

    TStringBuf Get(ui32 id) const {
        return Strings_[id];
    }

    template<typename F>
    void ForEach(F&& f) const {
        for (TStringBuf s : Strings_) {
            f(s);
        }
    }

private:
    TMemoryPool Pool_;
    TVector<TStringBuf> Strings_;
    absl::flat_hash_map<TStringBuf, ui32> StringToId_;
};

// Common string IDs for rendering
struct TCommonStrings {
    ui32 Empty;
    ui32 All;
    ui32 Container;
    ui32 Process;
    ui32 Thread;
    ui32 Signal;
    ui32 Native;
    ui32 Kernel;
    ui32 Python;
    ui32 Php;
    ui32 UnsymbolizedFunction;
    ui32 UnknownMapping;
    ui32 TruncatedStack;
    ui32 PrunedStack;
    ui32 Samples;
    ui32 Function;
    ui32 UnsymbolizedAddress;  // "??"

    static TCommonStrings Build(TStringInterner& table) {
        return TCommonStrings{
            .Empty = table.Intern(""),
            .All = table.Intern("all"),
            .Container = table.Intern("container"),
            .Process = table.Intern("process"),
            .Thread = table.Intern("thread"),
            .Signal = table.Intern("signal"),
            .Native = table.Intern("native"),
            .Kernel = table.Intern("kernel"),
            .Python = table.Intern("python"),
            .Php = table.Intern("php"),
            .UnsymbolizedFunction = table.Intern("<unsymbolized function>"),
            .UnknownMapping = table.Intern("<unknown mapping>"),
            .TruncatedStack = table.Intern("(truncated stack)"),
            .PrunedStack = table.Intern("(pruned stack)"),
            .Samples = table.Intern("samples"),
            .Function = table.Intern("Function"),
            .UnsymbolizedAddress = table.Intern("??"),
        };
    }

    ui32 GetOriginId(TStringBuf binaryPath) const {
        if (binaryPath == "[kernel]") {
            return Kernel;
        } else if (binaryPath == "[python]") {
            return Python;
        } else if (binaryPath == "[php]") {
            return Php;
        }
        return Native;
    }
};

// Rendered node data
struct TRenderedNode {
    ui32 NameId;
    ui32 FileId;
    ui32 OriginId;
    ui32 KindId;
};

// Node renderer - converts raw IDs to interned strings
// Derives NodeKind from label keys at render time
class TNodeRenderer {
public:
    TNodeRenderer(
        TProfile& profile,
        TStringInterner& stringTable,
        const TCommonStrings& common,
        const TLabelKeyIds& keyIds,
        const NProto::NProfile::RenderOptions& options
    )
        : Profile_(profile)
        , StringTable_(stringTable)
        , Common_(common)
        , KeyIds_(keyIds)
        , Options_(options)
    {
        // Dense stringId -> output-id cache: a flat array indexed by the
        // profile's internal string index, replacing a hash map. One cache miss
        // per first-touch, none afterwards, and no hashing/probing on the hot path.
        StringIdCache_.assign(Profile_.Strings().size(), NotInterned);
    }

    // Intern a profile string, caching by dense string index to avoid rehashing on repeats.
    ui32 InternProfileString(TProfileString str) {
        TStringId id = str.GetIndex();
        if (!id.IsValid()) {
            return Common_.Empty;
        }
        ui32& slot = StringIdCache_[id.GetInternalIndex()];
        if (slot == NotInterned) {
            slot = StringTable_.InternStable(str.View());
        }
        return slot;
    }

    TRenderedNode Render(const TFlameNodeId& identity) {
        // Use index-based switch instead of std::visit to avoid lambda object creation overhead
        switch (identity.index()) {
            case 0:  // std::monostate (root)
                return TRenderedNode{
                    .NameId = Common_.All,
                    .FileId = Common_.Empty,
                    .OriginId = Common_.Empty,
                    .KindId = Common_.Empty,
                };
            case 1:  // TLabelId
                return RenderLabel(std::get<1>(identity));
            case 2:  // TFrameKey
                return RenderFrame(std::get<2>(identity));
            default:
                Y_UNREACHABLE();
        }
    }

private:
    TRenderedNode RenderLabel(TLabelId labelId) {
        TRenderedNode result{
            .NameId = Common_.Empty,
            .FileId = Common_.Empty,
            .OriginId = Common_.Empty,
            .KindId = Common_.Empty,
        };

        TLabel label = Profile_.Label(labelId);
        TStringId keyId = label.GetKey().GetIndex();
        auto labelType = KeyIds_.GetLabel(keyId);

        if (!labelType) {
            return result;
        }

        switch (*labelType) {
            case NProto::NProfile::Workload:
                result.NameId = InternProfileString(label.GetString());
                result.KindId = Common_.Container;
                break;
            case NProto::NProfile::ProcessId:
                result.NameId = StringTable_.Intern(ToString(label.GetNumber()));
                result.KindId = Common_.Process;
                break;
            case NProto::NProfile::ProcessCommand:
                result.NameId = InternProfileString(label.GetString());
                result.KindId = Common_.Process;
                break;
            case NProto::NProfile::ThreadId:
                result.NameId = StringTable_.Intern(ToString(label.GetNumber()));
                result.KindId = Common_.Thread;
                break;
            case NProto::NProfile::ThreadCommand:
                result.NameId = InternProfileString(label.GetString());
                result.KindId = Common_.Thread;
                break;
            case NProto::NProfile::SignalName:
                result.NameId = InternProfileString(label.GetString());
                result.KindId = Common_.Signal;
                break;
            default:
                break;
        }

        return result;
    }

    TRenderedNode RenderFrame(const TFrameKey& frame) {
        TRenderedNode result{
            .NameId = Common_.Empty,
            .FileId = Common_.Empty,
            .OriginId = Common_.Empty,
            .KindId = Common_.Empty,
        };

        // Handle truncated stack marker
        if (frame.IsTruncatedStack()) {
            result.NameId = Common_.TruncatedStack;
            return result;
        }

        TStringBuf binaryPath = frame.BinaryId != TBinaryId::Zero()
            ? Profile_.Binary(frame.BinaryId).GetPath().View()
            : TStringBuf{};

        if (frame.NameId.IsValid()) {
            TProfileString name = Profile_.String(frame.NameId);
            if (!IsInvalidFunctionName(name.View())) {
                result.NameId = InternProfileString(name);
            }
        }

        if (Options_.show_file_names() && frame.FileId.IsValid()) {
            TStringBuf file = SanitizeFileName(Profile_.String(frame.FileId).View());
            if (!IsInvalidFilename(file)) {
                FileBuffer_.clear();
                FileBuffer_ += Options_.file_path_prefix();
                FileBuffer_ += file;
                if (Options_.show_line_numbers() && frame.Line > 0) {
                    FileBuffer_ += ':';
                    FileBuffer_ += ToString(frame.Line);
                }
                result.FileId = StringTable_.Intern(FileBuffer_);
            }
        }

        if (result.NameId == Common_.Empty) {
            if (frame.BinaryId == TBinaryId::Zero()) {
                result.NameId = Common_.UnknownMapping;
            } else if (!frame.NameId.IsValid()) {
                result.NameId = Common_.UnsymbolizedAddress;
            } else {
                result.NameId = Common_.UnsymbolizedFunction;
            }
        }

        result.OriginId = Common_.GetOriginId(binaryPath);
        return result;
    }

    TProfile& Profile_;
    TStringInterner& StringTable_;
    const TCommonStrings& Common_;
    const TLabelKeyIds& KeyIds_;
    const NProto::NProfile::RenderOptions& Options_;
    TString FileBuffer_;
    static constexpr ui32 NotInterned = std::numeric_limits<ui32>::max();
    TVector<ui32> StringIdCache_;  // Dense TStringId index → interned output ID
};

template <typename Writer, size_t N>
void WriteKey(Writer& writer, const char (&key)[N]) {
    writer.Key(key, N - 1);  // N includes null terminator
}

template <typename Writer>
void WriteNodeJson(
    Writer& writer,
    const TRenderedNode& rendered,
    i32 parentLevelIdx,
    i64 sampleCount,
    i64 eventCount
) {
    writer.StartObject();
    WriteKey(writer, "parentIndex");
    writer.Int(parentLevelIdx);
    WriteKey(writer, "textId");
    writer.Uint(rendered.NameId);
    WriteKey(writer, "sampleCount");
    writer.Int64(sampleCount);
    WriteKey(writer, "eventCount");
    writer.Int64(eventCount);
    WriteKey(writer, "frameOrigin");
    writer.Uint(rendered.OriginId);
    WriteKey(writer, "kind");
    writer.Uint(rendered.KindId);
    WriteKey(writer, "file");
    writer.Uint(rendered.FileId);
    writer.EndObject();
}

void RenderTrieToJson(
    const TFlameTrieResult& built,
    TProfile& profile,
    const TLabelKeyIds& keyIds,
    IOutputStream& out,
    const NProto::NProfile::RenderOptions& options,
    ui32 sampleTypeIndex
) {
    const TFlameTrie& trie = built.Trie;
    const TVector<TFlameNodeId>& denseToNode = built.DenseToNode;
    TStringInterner stringTable;
    TCommonStrings common = TCommonStrings::Build(stringTable);

    TNodeRenderer renderer(profile, stringTable, common, keyIds, options);

    // Resolve each distinct identity once, on first encounter. Many trie nodes share an
    // identity (same frame in different call paths), so this avoids re-resolving it, and the
    // first-encounter order keeps string interning in BFS order (output ids stay stable).
    // A slot is empty until rendered; no real NameId is ever NotRendered (max ui32).
    static constexpr ui32 NotRendered = std::numeric_limits<ui32>::max();
    TVector<TRenderedNode> renderedByDense(denseToNode.size(), TRenderedNode{.NameId = NotRendered});
    auto renderDense = [&](ui32 d) -> const TRenderedNode& {
        TRenderedNode& slot = renderedByDense[d];
        if (slot.NameId == NotRendered) {
            slot = renderer.Render(denseToNode[d]);
        }
        return slot;
    };

    // Calculate minWeight threshold based on root event count
    const i64 rootEventCount = trie.Root().GetValue().EventCount;
    const i64 minEventThreshold = options.min_weight() > 0.0
        ? static_cast<i64>(rootEventCount * options.min_weight())
        : 0;

    // Pre-intern "(pruned stack)" for min-weight filtered children aggregation
    const ui32 prunedNameId = minEventThreshold > 0
        ? stringTable.Intern("(pruned stack)")
        : 0;

    rapidjson::StringBuffer buffer;
    rapidjson::Writer<rapidjson::StringBuffer> writer(buffer);

    writer.StartObject();
    WriteKey(writer, "rows");
    writer.StartArray();

    // Level-order BFS traversal.
    // Each node is rendered exactly once (when first seen as a child, or the
    // root) and the TRenderedNode is carried through the BFS, so writing never
    // re-renders. Previously every node was rendered twice (once to sort it among
    // its siblings, once to write it as an entry).
    constexpr ui32 PrunedNodeMarker = std::numeric_limits<ui32>::max();

    struct TLevelEntry {
        ui32 NodeIdx;            // PrunedNodeMarker => aggregated "(pruned stack)" node
        i32 ParentLevelIdx;
        TRenderedNode Rendered;  // pre-rendered, never recomputed
        i64 SampleCount;
        i64 EventCount;
    };

    TVector<TLevelEntry> currentLevel;
    TVector<TLevelEntry> nextLevel;
    TVector<TLevelEntry> children;  // reused per node for sibling sorting

    {
        auto root = trie.Root();
        const auto& rootValue = root.GetValue();
        currentLevel.push_back({0, -1, renderDense(root.GetKey()),
                                rootValue.SampleCount, rootValue.EventCount});
    }

    static constexpr TStringBuf prunedStackName = "(pruned stack)";

    while (!currentLevel.empty()) {
        writer.StartArray();

        nextLevel.clear();

        for (size_t levelIdx = 0; levelIdx < currentLevel.size(); ++levelIdx) {
            const auto& entry = currentLevel[levelIdx];

            WriteNodeJson(writer, entry.Rendered, entry.ParentLevelIdx,
                         entry.SampleCount, entry.EventCount);

            if (entry.NodeIdx == PrunedNodeMarker) {
                continue;  // Pruned nodes have no children
            }

            auto node = trie.NodeAt(entry.NodeIdx);
            children.clear();
            i64 prunedSampleCount = 0;
            i64 prunedEventCount = 0;
            ui32 prunedOriginId = common.Native;  // Default, will be set to first pruned child's origin

            for (auto child = node.GetFirstChild(); !child.IsZero(); child = child.GetNextSibling()) {
                const auto& childValue = child.GetValue();
                if (minEventThreshold > 0 && childValue.EventCount < minEventThreshold) {
                    // Track origin from the first pruned child
                    if (prunedEventCount == 0) {
                        prunedOriginId = renderDense(child.GetKey()).OriginId;
                    }
                    prunedSampleCount += childValue.SampleCount;
                    prunedEventCount += childValue.EventCount;
                    continue;
                }
                // Render child once; it is carried unchanged into the next level.
                children.push_back({child.GetId(), static_cast<i32>(levelIdx),
                                    renderDense(child.GetKey()),
                                    childValue.SampleCount, childValue.EventCount});
            }

            // 90% of nodes have <=1 child — nothing to sort.
            if (children.size() > 1) std::sort(children.begin(), children.end(), [&](const TLevelEntry& a, const TLevelEntry& b) {
                TStringBuf nameA = stringTable.Get(a.Rendered.NameId);
                TStringBuf nameB = stringTable.Get(b.Rendered.NameId);
                return nameA != nameB
                    ? nameA < nameB
                    : stringTable.Get(a.Rendered.FileId) < stringTable.Get(b.Rendered.FileId);
            });

            // "(pruned stack)" should be sorted alphabetically among siblings.
            auto pushPruned = [&] {
                nextLevel.push_back({
                    PrunedNodeMarker,
                    static_cast<i32>(levelIdx),
                    TRenderedNode{
                        .NameId = prunedNameId,
                        .FileId = 0,  // empty
                        .OriginId = prunedOriginId,
                        .KindId = common.Function,
                    },
                    prunedSampleCount,
                    prunedEventCount,
                });
            };

            bool prunedAdded = false;
            for (auto& child : children) {
                if (!prunedAdded && prunedEventCount > 0 &&
                    prunedStackName < stringTable.Get(child.Rendered.NameId))
                {
                    pushPruned();
                    prunedAdded = true;
                }
                nextLevel.push_back(std::move(child));
            }

            if (!prunedAdded && prunedEventCount > 0) {
                pushPruned();
            }
        }

        writer.EndArray();
        currentLevel.swap(nextLevel);
    }

    writer.EndArray();

    // Intern eventType BEFORE writing stringTable (so it's included in the output)
    TValueType valueType = profile.ValueType(TValueTypeId::FromInternalIndex(sampleTypeIndex));
    TString eventType = TString{valueType.GetType().View()} + "." + valueType.GetUnit().View();
    ui32 eventTypeId = stringTable.Intern(eventType);

    WriteKey(writer, "stringTable");
    writer.StartArray();
    stringTable.ForEach([&writer](TStringBuf s) {
        writer.String(s.data(), s.size());
    });
    writer.EndArray();

    WriteKey(writer, "meta");
    writer.StartObject();
    WriteKey(writer, "version");
    writer.Int(2);
    WriteKey(writer, "eventType");
    writer.Uint(eventTypeId);
    WriteKey(writer, "frameType");
    writer.Uint(common.Function);
    writer.EndObject();

    writer.EndObject();

    out.Write(buffer.GetString(), buffer.GetSize());
}

} // namespace

////////////////////////////////////////////////////////////////////////////////

void RenderFlameGraphJson(
    const NProto::NProfile::Profile& profile,
    IOutputStream& out,
    const NProto::NProfile::RenderOptions& options
) {
    // Reshape the input into render form: strip garbage root frames, sanitize thread
    // names, and (unless line numbers are requested) collapse source locations.
    NProto::NProfile::MergeOptions mergeOptions;
    mergeOptions.set_strip_garbage_root_frames(true);
    mergeOptions.set_sanitize_thread_names(true);
    mergeOptions.set_ignore_source_locations(!options.show_line_numbers());

    NProto::NProfile::Profile merged;
    MergeProfiles({profile}, &merged, mergeOptions);

    TProfile mergedProfile{&merged};
    auto keyIds = TLabelKeyIds::Build(mergedProfile);
    ui32 sampleTypeIndex = ResolveSampleTypeIndex(mergedProfile);

    auto built = BuildFlameTrie(mergedProfile, keyIds, options, sampleTypeIndex);
    RenderTrieToJson(built, mergedProfile, keyIds, out, options, sampleTypeIndex);
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
