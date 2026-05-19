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

using TFlameTrie = TTrie<TFlameNodeId, TFlameValue>;

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

TFlameTrie BuildFlameTrie(
    TProfile& profile,
    const TLabelKeyIds& keyIds,
    const NProto::NProfile::RenderOptions& options,
    ui32 sampleTypeIndex
) {
    TFlameTrie trie;

    TVector<TFlameValue> keyValues(profile.SampleKeys().size());
    for (auto sample : profile.Samples()) {
        ui32 keyIndex = sample.GetKey().GetIndex().GetInternalIndex();
        keyValues[keyIndex].SampleCount += 1;
        keyValues[keyIndex].EventCount += sample.GetValues()[sampleTypeIndex].GetValue();
    }

    // Process each sample key (unique stack)
    for (auto sampleKey : profile.SampleKeys()) {
        ui32 keyIndex = sampleKey.GetIndex().GetInternalIndex();
        TFlameValue value = keyValues[keyIndex];

        if (value.SampleCount == 0) {
            continue;
        }

        auto node = trie.Root();
        node.GetValue() += value;
        ui32 depth = 0;    // Track depth for truncation (includes labels, like Go)

        // Helper to descend and accumulate value
        auto descend = [&](TFlameNodeId id) {
            node = node.GetOrCreateChild(id);
            node.GetValue() += value;
            ++depth;
        };

        // Process labels (order matches Go renderer: containers, pid, process_name, thread_name, signal)
        // Note: Go does NOT include ThreadId (tid), only ThreadCommand (thread name)
        bool hasFirstContainer = false;
        std::array<TLabelId, 4> labelNodes{{
            TLabelId::Invalid(),  // ProcessId (pid)
            TLabelId::Invalid(),  // ProcessCommand (process name)
            TLabelId::Invalid(),  // ThreadCommand (thread name)
            TLabelId::Invalid(),  // SignalName
        }};

        for (auto label : sampleKey.GetLabels()) {
            TStringId keyId = label.GetKey().GetIndex();
            auto labelType = keyIds.GetLabel(keyId);

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
                        descend(TFlameNodeId{label.GetIndex()});
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
                    // Skip ThreadId - Go renderer doesn't include it
                    break;
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
                descend(TFlameNodeId{labelId});
            }
        }

        // Process stacks (reverse order - TProfile stores leaf-to-root, we need root-to-leaf)
        bool truncated = false;
        for (TStack stack : sampleKey.GetStacks() | std::views::reverse) {
            if (truncated) {
                break;
            }
            for (TStackFrame frame : stack.GetFrames() | std::views::reverse) {
                if (truncated) {
                    break;
                }
                TBinaryId binaryId = frame.GetBinary().GetIndex();
                TStringId binaryPathId = binaryId != TBinaryId::Zero()
                    ? frame.GetBinary().GetPath().GetIndex()
                    : TStringId::Invalid();

                TInlineChain chain = frame.GetInlineChain();
                const bool showFiles = options.show_file_names();
                const bool showLines = options.show_line_numbers();

                if (chain.GetLines().empty()) {
                    if (options.max_depth() > 0 && depth >= options.max_depth()) {
                        truncated = true;
                        break;
                    }
                    descend(TFlameNodeId{TFrameKey{
                        .NameId = TStringId::Invalid(),
                        .FileId = showFiles ? binaryPathId : TStringId::Invalid(),
                        .BinaryId = binaryId,
                        .Line = 0,
                        .Column = 0,
                    }});
                } else {
                    // TODO: Inline chains are stored in WRONG ORDER in TProfile.
                    // The pprof format stores inline chains in leaf-to-root order (innermost function first),
                    // and our pprof converters copy this order directly into TProfile without reversing.
                    // For correct flamegraph rendering (root-to-leaf), we should either:
                    //   1. Fix pprof converters to reverse inline chains when building TProfile
                    //   2. Fix this loop to iterate in reverse via `chain.GetLines() | std::views::reverse`
                    // Currently this renders inline chains in the wrong order (leaf-to-root instead of root-to-leaf).
                    for (TSourceLine line : chain.GetLines()) {
                        if (options.max_depth() > 0 && depth >= options.max_depth()) {
                            truncated = true;
                            break;
                        }
                        TFunction func = line.GetFunction();
                        TStringId fileId = func.GetFileName().GetIndex();
                        if (IsInvalidFilename(func.GetFileName().View())) {
                            fileId = binaryPathId;
                        }
                        descend(TFlameNodeId{TFrameKey{
                            .NameId = func.GetName().GetIndex(),
                            .FileId = showFiles ? fileId : TStringId::Invalid(),
                            .BinaryId = binaryId,
                            .Line = showLines ? line.GetLine() : 0,
                            .Column = showLines ? line.GetColumn() : 0,
                        }});
                    }
                }
            }
        }

        // Add truncated stack node if we hit the depth limit
        if (truncated) {
            descend(TFlameNodeId{TFrameKey::TruncatedStack()});
        }
    }

    trie.Finalize();
    return trie;
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
        FileBuffer_.reserve(256);
    }

    // Intern a profile string, caching the result to avoid rehashing
    ui32 InternProfileString(TStringId stringId) {
        if (!stringId.IsValid()) {
            return Common_.Empty;
        }
        auto [it, inserted] = StringIdCache_.try_emplace(stringId, 0);
        if (inserted) {
            // First time seeing this string - intern it (stable, no copy needed)
            it->second = StringTable_.InternStable(Profile_.String(stringId).View());
        }
        return it->second;
    }

    // Intern a profile string when we already have the view (avoids double lookup)
    ui32 InternProfileString(TStringId stringId, TStringBuf view) {
        auto [it, inserted] = StringIdCache_.try_emplace(stringId, 0);
        if (inserted) {
            it->second = StringTable_.InternStable(view);
        }
        return it->second;
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
                result.NameId = InternProfileString(label.GetString().GetIndex(), label.GetString().View());
                result.KindId = Common_.Container;
                break;
            case NProto::NProfile::ProcessId:
                result.NameId = StringTable_.Intern(ToString(label.GetNumber()));
                result.KindId = Common_.Process;
                break;
            case NProto::NProfile::ProcessCommand:
                result.NameId = InternProfileString(label.GetString().GetIndex(), label.GetString().View());
                result.KindId = Common_.Process;
                break;
            case NProto::NProfile::ThreadId:
                result.NameId = StringTable_.Intern(ToString(label.GetNumber()));
                result.KindId = Common_.Thread;
                break;
            case NProto::NProfile::ThreadCommand:
                result.NameId = InternProfileString(label.GetString().GetIndex(), label.GetString().View());
                result.KindId = Common_.Thread;
                break;
            case NProto::NProfile::SignalName:
                result.NameId = InternProfileString(label.GetString().GetIndex(), label.GetString().View());
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
            TStringBuf name = Profile_.String(frame.NameId).View();
            if (!IsInvalidFunctionName(name)) {
                result.NameId = InternProfileString(frame.NameId, name);
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
    absl::flat_hash_map<TStringId, ui32> StringIdCache_;  // Cache TStringId → interned ID
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
    const TFlameTrie& trie,
    TProfile& profile,
    const TLabelKeyIds& keyIds,
    IOutputStream& out,
    const NProto::NProfile::RenderOptions& options,
    ui32 sampleTypeIndex
) {
    TStringInterner stringTable;
    TCommonStrings common = TCommonStrings::Build(stringTable);

    TNodeRenderer renderer(profile, stringTable, common, keyIds, options);

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
    buffer.Reserve(trie.NodeCount() * 100);  // ~100 bytes per node estimate
    rapidjson::Writer<rapidjson::StringBuffer> writer(buffer);

    writer.StartObject();
    WriteKey(writer, "rows");
    writer.StartArray();

    // Level-order BFS traversal
    // Each entry: (nodeIdx, parentLevelIdx) where nodeIdx=UINT32_MAX means pruned stack
    constexpr ui32 PrunedNodeMarker = std::numeric_limits<ui32>::max();

    struct TLevelEntry {
        ui32 NodeIdx;
        i32 ParentLevelIdx;
        i64 SampleCount;  // Only used for pruned nodes
        i64 EventCount;   // Only used for pruned nodes
        ui32 OriginId;    // Only used for pruned nodes
    };

    // Child info for sorting - holds node index and pre-rendered data
    struct TChildInfo {
        ui32 NodeIdx;
        TRenderedNode Rendered;
    };

    TVector<TLevelEntry> currentLevel;
    TVector<TLevelEntry> nextLevel;
    TVector<TChildInfo> children;

    currentLevel.push_back({0, -1, 0, 0, 0});  // Root has parent -1

    while (!currentLevel.empty()) {
        writer.StartArray();

        nextLevel.clear();

        for (size_t levelIdx = 0; levelIdx < currentLevel.size(); ++levelIdx) {
            const auto& entry = currentLevel[levelIdx];

            // Check if this is a pruned stack node (min-weight filtered)
            if (entry.NodeIdx == PrunedNodeMarker) {
                TRenderedNode prunedNode{
                    .NameId = prunedNameId,
                    .FileId = 0,  // empty
                    .OriginId = entry.OriginId,
                    .KindId = common.Function,
                };
                WriteNodeJson(writer, prunedNode, entry.ParentLevelIdx,
                             entry.SampleCount, entry.EventCount);
                continue;  // Pruned nodes have no children
            }

            auto node = trie.NodeAt(entry.NodeIdx);
            const auto& nodeValue = node.GetValue();
            // Render current node directly (no caching needed - each node visited once)
            WriteNodeJson(writer, renderer.Render(node.GetKey()), entry.ParentLevelIdx,
                         nodeValue.SampleCount, nodeValue.EventCount);

            children.clear();
            i64 prunedSampleCount = 0;
            i64 prunedEventCount = 0;
            ui32 prunedOriginId = common.Native;  // Default, will be set to first pruned child's origin

            for (auto child = node.GetFirstChild(); !child.IsZero(); child = child.GetNextSibling()) {
                const auto& childValue = child.GetValue();
                if (minEventThreshold > 0 && childValue.EventCount < minEventThreshold) {
                    // Track origin from first pruned child (like Go does)
                    if (prunedEventCount == 0) {
                        prunedOriginId = renderer.Render(child.GetKey()).OriginId;
                    }
                    prunedSampleCount += childValue.SampleCount;
                    prunedEventCount += childValue.EventCount;
                    continue;
                }
                // Render child once and store for sorting/output
                children.push_back({child.GetId(), renderer.Render(child.GetKey())});
            }

            std::sort(children.begin(), children.end(), [&](const TChildInfo& a, const TChildInfo& b) {
                TStringBuf nameA = stringTable.Get(a.Rendered.NameId);
                TStringBuf nameB = stringTable.Get(b.Rendered.NameId);
                return nameA != nameB
                    ? nameA < nameB
                    : stringTable.Get(a.Rendered.FileId) < stringTable.Get(b.Rendered.FileId);
            });

            // "(pruned stack)" should be sorted alphabetically among siblings
            static constexpr TStringBuf prunedStackName = "(pruned stack)";
            bool prunedAdded = false;
            for (const auto& child : children) {
                if (!prunedAdded && prunedEventCount > 0 &&
                    prunedStackName < stringTable.Get(child.Rendered.NameId))
                {
                    nextLevel.push_back({
                        PrunedNodeMarker,
                        static_cast<i32>(levelIdx),
                        prunedSampleCount,
                        prunedEventCount,
                        prunedOriginId,
                    });
                    prunedAdded = true;
                }
                nextLevel.push_back({child.NodeIdx, static_cast<i32>(levelIdx), 0, 0, 0});
            }

            if (!prunedAdded && prunedEventCount > 0) {
                nextLevel.push_back({
                    PrunedNodeMarker,
                    static_cast<i32>(levelIdx),
                    prunedSampleCount,
                    prunedEventCount,
                    prunedOriginId,
                });
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
    NProto::NProfile::MergeOptions mergeOptions;
    mergeOptions.set_strip_garbage_root_frames(true);
    mergeOptions.set_sanitize_thread_names(true);
    mergeOptions.set_ignore_source_locations(!options.show_line_numbers());

    NProto::NProfile::Profile merged;
    MergeProfiles({profile}, &merged, mergeOptions);

    TProfile mergedProfile{&merged};
    auto keyIds = TLabelKeyIds::Build(mergedProfile);
    ui32 sampleTypeIndex = ResolveSampleTypeIndex(mergedProfile);
    auto trie = BuildFlameTrie(mergedProfile, keyIds, options, sampleTypeIndex);
    RenderTrieToJson(trie, mergedProfile, keyIds, out, options, sampleTypeIndex);
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
