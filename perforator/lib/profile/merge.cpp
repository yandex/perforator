#include "builder.h"
#include "entity_index.h"
#include "merge.h"
#include "profile.h"

#include <contrib/libs/re2/re2/re2.h>

#include <library/cpp/containers/absl/flat_hash_map.h>
#include <library/cpp/containers/absl/flat_hash_set.h>

#include <util/digest/numeric.h>
#include <util/generic/algorithm.h>

#include <bitset>
#include <cctype>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

template <CStrongIndex Index>
class TIndexRemapping {
public:
    TIndexRemapping(i32 count)
        : Mapping_(count, Index::Invalid())
    {}

    [[nodiscard]] Index Map(Index prev) const {
        return At(prev);
    }

    void Set(Index from, Index to) {
        Index& prev = At(from);
        if (prev.IsValid()) {
            Y_ENSURE(false, "Duplicate index " << *from);
        } else {
            Y_ENSURE(to.IsValid());
            prev = to;
        }
    }

    [[nodiscard]] Index TryMap(Index from, TFunctionRef<Index()> calcer) {
        if (Index idx = Map(from); idx.IsValid()) {
            return idx;
        }

        Index to = calcer();
        Set(from, to);

        return to;
    }

private:
    const Index& At(Index idx) const {
        ui32 pos = idx.GetInternalIndex();
        Y_ENSURE(pos < Mapping_.size());
        return Mapping_[pos];
    }

    Index& At(Index idx) {
        ui32 pos = idx.GetInternalIndex();
        Y_ENSURE(pos < Mapping_.size());
        return Mapping_[pos];
    }

private:
    TVector<Index> Mapping_;
};

class TMergePolicy {
    static constexpr ui64 MaxRequiredLabelCount = 256;

public:
    TMergePolicy(TProfile profile, const NProto::NProfile::MergeOptions& options)
        : Profile_{profile}
        , Options_{options}
    {
        PopulateFilters();
    }

    ui64 SamplePeriod() const {
        return Max<ui64>(Options_.sample_period(), 1);
    }

    bool MergeBySymbolizedNames() const {
        return Options_.merge_by_symbolized_names();
    }

    bool IgnoreBinaryPaths() const {
        return Options_.ignore_binary_paths();
    }

    bool IgnoreTimestamps() const {
        return Options_.ignore_timestamps();
    }

    bool IgnoreSourceLocations() const {
        return Options_.ignore_source_locations();
    }

    bool StripGarbageRootFrames() const {
        return Options_.strip_garbage_root_frames();
    }

    bool SanitizeThreadNames() const {
        return Options_.sanitize_thread_names();
    }

    bool AllowSample(TSample sample) const {
        if (HasTriviallyPositiveSampleFilter_) {
            return true;
        }
        if (HasTriviallyNegativeSampleFilter_) {
            return false;
        }

        return true
            && SampleHasOneOfRequiredBinaries(sample)
            && SampleHasAllOfRequiredLabels(sample);
    }

    bool AllowLabel(TLabel label) const {
        return (PreservedLabelKeys_.empty() || PreservedLabelKeys_.contains(label.GetKey().GetIndex()))
            && !DroppedLabelKeys_.contains(label.GetKey().GetIndex());
    }

    bool AllowValueType(TValueType valueType) const {
        return AllowedValueTypes_.empty() || AllowedValueTypes_.contains(valueType.GetIndex());
    }

private:
    bool SampleHasOneOfRequiredBinaries(TSample sample) const {
        if (RequiredOneOfBinaries_.empty()) {
            return true;
        }

        TSampleKey key = sample.GetKey();
        for (TStack stack : key.GetStacks()) {
            for (TStackFrame frame : stack.GetFrames()) {
                TBinaryId binaryId = frame.GetBinary().GetIndex();
                if (RequiredOneOfBinaries_.contains(binaryId)) {
                    return true;
                }
            }
        }
        return false;
    }

    bool SampleHasAllOfRequiredLabels(TSample sample) const {
        if (RequiredAllOfLabels_.empty()) {
            return true;
        }

        std::bitset<MaxRequiredLabelCount> found;
        Y_ASSERT(found.size() >= RequiredAllOfLabels_.size());

        TSampleKey key = sample.GetKey();
        for (TLabel label : key.GetLabels()) {
            auto it = RequiredAllOfLabels_.find(label.GetIndex());
            if (it != RequiredAllOfLabels_.end()) {
                found.set(it->second);
            }
        }

        return found.count() == RequiredAllOfLabels_.size();
    }

    void PopulateFilters() {
        PopulateSampleFilters();
        PopulateLabelFilters();
        PopulateValueTypeFilters();
    }

    void PopulateSampleFilters() {
        HasTriviallyPositiveSampleFilter_ = !Options_.has_sample_filter();
        if (HasTriviallyPositiveSampleFilter_) {
            return;
        }

        NProto::NProfile::SampleFilter filter = Options_.sample_filter();

        // Map binaries to internal ids.
        absl::flat_hash_set<TStringBuf> buildIds{
            filter.required_one_of_build_ids().begin(),
            filter.required_one_of_build_ids().end(),
        };
        for (TBinary binary : Profile_.Binaries()) {
            TStringBuf buildId = binary.GetBuildId().View();
            if (buildIds.erase(buildId)) {
                RequiredOneOfBinaries_.insert(binary.GetIndex());
                if (buildIds.empty()) {
                    break;
                }
            }
        }
        if (RequiredOneOfBinaries_.empty() && !buildIds.empty()) {
            HasTriviallyNegativeSampleFilter_ = true;
        }

        // Map labels to internal ids.
        ui64 labelCount = filter.required_all_of_string_labels_size()
            + filter.required_all_of_numeric_labels_size();
        Y_ENSURE(
            labelCount <= MaxRequiredLabelCount,
            "Too many required labels, only " << MaxRequiredLabelCount
                << " allowed, got " << labelCount
        );

        for (TLabel label : Profile_.Labels()) {
            if (label.IsNumber()) {
                MatchLabelFilter(label, label.GetNumber(), filter.mutable_required_all_of_numeric_labels());
            } else {
                MatchLabelFilter(label, label.GetString(), filter.mutable_required_all_of_string_labels());
            }
        }

        // If we cannot find some labels, we will not be able to accept any sample.
        if (RequiredAllOfLabels_.size() != labelCount) {
            HasTriviallyNegativeSampleFilter_ = true;
        }

        if (labelCount == 0 && buildIds.empty()) {
            HasTriviallyPositiveSampleFilter_ = true;
        }
    }

    template <typename Value, typename ProtoMap>
    void MatchLabelFilter(TLabel label, Value value, ProtoMap* map) {
        auto it = map->find(label.GetKey().View());
        if (it == map->end()) {
            return;
        }

        if (value == it->second) {
            ui32 id = RequiredAllOfLabels_.size();
            RequiredAllOfLabels_.try_emplace(label.GetIndex(), id);
        }
    }

    void PopulateLabelFilters() {
        for (TStringBuf show : Options_.label_filter().keys_show()) {
            for (TLabel label : Profile_.Labels()) {
                if (label.GetKey().View() == show) {
                    PreservedLabelKeys_.insert(label.GetKey().GetIndex());
                }
            }
        }
        for (TStringBuf hide : Options_.label_filter().keys_hide()) {
            for (TLabel label : Profile_.Labels()) {
                if (label.GetKey().View() == hide) {
                    DroppedLabelKeys_.insert(label.GetKey().GetIndex());
                }
            }
        }
        for (TStringBuf showRegex : Options_.label_filter().keys_show_regex()) {
            re2::RE2 regex{showRegex};
            for (TLabel label : Profile_.Labels()) {
                if (regex.PartialMatch(label.GetKey().View(), regex)) {
                    PreservedLabelKeys_.insert(label.GetKey().GetIndex());
                }
            }
        }
        for (TStringBuf hideRegex : Options_.label_filter().keys_hide_regex()) {
            re2::RE2 regex{hideRegex};
            for (TLabel label : Profile_.Labels()) {
                if (regex.PartialMatch(label.GetKey().View(), regex)) {
                    DroppedLabelKeys_.insert(label.GetKey().GetIndex());
                }
            }
        }
    }

    void PopulateValueTypeFilters() {
        if (Options_.value_type_filter().allowlist().empty()) {
            return;
        }

        for (TValueType valueType : Profile_.ValueTypes()) {
            for (TStringBuf allowedValueTypeName : Options_.value_type_filter().allowlist()) {
                if (allowedValueTypeName.SkipPrefix(valueType.GetType().View()) &&
                    allowedValueTypeName.SkipPrefix(".") &&
                    allowedValueTypeName == valueType.GetUnit().View()
                ) {
                    AllowedValueTypes_.insert(valueType.GetIndex());
                }
            }
        }
    }

private:
    TProfile Profile_;
    const NProto::NProfile::MergeOptions& Options_;

    bool HasTriviallyPositiveSampleFilter_ = false;
    bool HasTriviallyNegativeSampleFilter_ = false;
    absl::flat_hash_map<TLabelId, ui32> RequiredAllOfLabels_;
    absl::flat_hash_set<TBinaryId> RequiredOneOfBinaries_;
    absl::flat_hash_set<TStringId> PreservedLabelKeys_;
    absl::flat_hash_set<TStringId> DroppedLabelKeys_;
    absl::flat_hash_set<TValueTypeId> AllowedValueTypes_;
};

static i32 LabelIndexSpaceSize(TProfile profile) {
    auto labels = profile.Labels();
    return labels.empty() ? 0 : labels.back().GetIndex().GetInternalIndex() + 1;
}

class TSingleProfileMerger {
public:
    TSingleProfileMerger(
        TProfileBuilder& builder,
        const NProto::NProfile::MergeOptions& options,
        TProfile profile,
        ui32 profileIndex
    )
        : Builder_{builder}
        , Profile_{profile}
        , IsFirstProfile_{profileIndex == 0}
        , Policy_{profile, options}
        , SamplingCounter_(NumericHash(profileIndex) % Policy_.SamplePeriod())
        , Strings_{static_cast<i32>(Profile_.Strings().size())}
        , ValueTypes_{static_cast<i32>(Profile_.ValueTypes().size())}
        , Samples_{static_cast<i32>(Profile_.Samples().size())}
        , SampleKeys_{static_cast<i32>(Profile_.SampleKeys().size())}
        , Stacks_{static_cast<i32>(Profile_.Stacks().size())}
        , Binaries_{static_cast<i32>(Profile_.Binaries().size())}
        , StackSegments_{static_cast<i32>(Profile_.StackSegments().size())}
        , StackFrames_{static_cast<i32>(Profile_.StackFrames().size())}
        , InlineChains_{static_cast<i32>(Profile_.InlineChains().size())}
        , SourceLines_{static_cast<i32>(Profile_.SourceLines().size())}
        , Functions_{static_cast<i32>(Profile_.Functions().size())}
        , LabelGroups_{static_cast<i32>(Profile_.LabelGroups().size())}
        , Labels_{LabelIndexSpaceSize(Profile_)}
    {
        if (Policy_.SanitizeThreadNames()) {
            for (const TString& key : GetAllWellKnownLabelKeys(NProto::NProfile::ThreadCommand)) {
                ThreadLabelKeyIds_.push_back(Builder_.AddString(key));
            }
        }
    }

    void Merge() {
        MergeMetadata();
        MergeBinaries();
        MergeSamples();
    }

private:
    void MergeMetadata() {
        if (!IsFirstProfile_) {
            return;
        }

        const auto& metadata = Profile_.GetMetadata();
        if (metadata.default_sample_type() == 0) {
            return;
        }

        TStringBuf defaultType = Profile_.String(TStringId::FromInternalIndex(metadata.default_sample_type())).View();
        for (TValueType valueType : Profile_.ValueTypes()) {
            if (Policy_.AllowValueType(valueType) && valueType.GetType().View() == defaultType) {
                Builder_.Metadata().GetProto().set_default_sample_type(*MapString(valueType.GetType()));
                return;
            }
        }
    }

    void MergeBinaries() {
        // Some tools consider first binary/mapping as main.
        // To improve stability we do something similar to how pprof merges profiles - main binary is inferred from first profile.
        if (IsFirstProfile_ && Profile_.Binaries().size() > 0) {
            [[maybe_unused]] auto _ = MapBinary(Profile_.Binary(TBinaryId::Zero()));
        }
    }

    void MergeSamples() {
        for (TSample sample : Profile_.Samples()) {
            if (Policy_.AllowSample(sample)) {
                if (SamplingCounter_++ % Policy_.SamplePeriod() == 0) {
                    MergeSample(sample);
                }
            }
        }
    }

    void MergeSample(TSample sample) {
        auto builder = Builder_.AddSample();

        if (auto ts = sample.GetProtoTimestamp(); ts && !Policy_.IgnoreTimestamps()) {
            builder.SetTimestamp(ts->seconds(), ts->nanos());
        }

        builder.SetSampleKey(MapSampleKey(sample.GetKey()));

        for (auto sampleValue : sample.GetValues()) {
            auto valueType = sampleValue.GetValueType();
            if (Policy_.AllowValueType(valueType)) {
                builder.AddValue(MapValueType(valueType), sampleValue.GetValue());
            }
        }

        builder.Finish();
    }

    TValueTypeId MapValueType(TValueType type) {
        return ValueTypes_.TryMap(type.GetIndex(), [&, this] {
            return Builder_.AddValueType(
                MapString(type.GetType()),
                MapString(type.GetUnit())
            );
        });
    }

    // Check if a frame is a "garbage root" frame (no mapping, no lines)
    static bool IsGarbageRootFrame(TStackFrame frame) {
        return frame.GetBinary().GetIndex() == TBinaryId::Zero()
            && frame.GetInlineChain().GetLines().empty();
    }

    TSampleKeyId MapSampleKey(TSampleKey key) {
        return SampleKeys_.TryMap(key.GetIndex(), [&key, this] {
            auto builder = Builder_.AddSampleKey();

            builder.SetLabelGroup(MapLabelGroup(key.GetLabelGroup()));

            auto stacks = key.GetStacks();
            const i32 stackCount = static_cast<i32>(stacks.size());
            if (Policy_.StripGarbageRootFrames() && stackCount > 0) {
                // Find first non-garbage frame from root side
                i32 stopStackIdx = stackCount;
                i32 stopFrameIdx = 0;

                for (i32 stackIdx = stackCount - 1; stackIdx >= 0 && stopStackIdx == stackCount; --stackIdx) {
                    TStack stack = stacks[stackIdx];
                    auto frames = stack.GetFrames();
                    const i32 frameCount = static_cast<i32>(frames.size());
                    for (i32 frameIdx = frameCount - 1; frameIdx >= 0; --frameIdx) {
                        if (!IsGarbageRootFrame(frames[frameIdx])) {
                            stopStackIdx = stackIdx;
                            stopFrameIdx = frameIdx;
                            break;
                        }
                    }
                }

                if (stopStackIdx < stackCount) {
                    for (TStack stack : stacks | std::views::take(stopStackIdx)) {
                        builder.AddStack(MapStack(stack));
                    }

                    TStack stack = stacks[stopStackIdx];
                    auto frames = stack.GetFrames();
                    if (stopFrameIdx < static_cast<i32>(frames.size()) - 1) {
                        builder.AddStack(MapStackPartial(stack, stopFrameIdx + 1));
                    } else {
                        builder.AddStack(MapStack(stack));
                    }
                }
            } else {
                for (TStack stack : key.GetStacks()) {
                    builder.AddStack(MapStack(stack));
                }
            }

            for (TLabel label : key.GetDirectLabels()) {
                if (Policy_.AllowLabel(label)) {
                    builder.AddLabel(MapLabel(label));
                }
            }

            return builder.Finish();
        });
    }

    TLabelGroupId MapLabelGroup(TLabelGroup labelGroup) {
        return LabelGroups_.TryMap(labelGroup.GetIndex(), [&labelGroup, this] {
            auto builder = Builder_.AddLabelGroup();

            for (auto&& label : labelGroup.GetLabels()) {
                if (Policy_.AllowLabel(label)) {
                    builder.AddLabel(MapLabel(label));
                }
            }

            return builder.Finish();
        });
    }

    TStringId SanitizeThreadName(TProfileString name) {
        TStringBuf str = name.View();

        // Chop trailing digits.
        size_t i = str.size();
        for (; i > 0; --i) {
            if (!std::isdigit(str[i - 1])) {
                break;
            }
        }

        // If there is no trailing digits, save hashmap lookup.
        if (i == str.size()) {
            return MapString(name);
        }

        return MapString(str.Head(i));
    }

    TLabelId MapLabel(TLabel label) {
        return Labels_.TryMap(label.GetIndex(), [&, this] {
            if (label.IsNumber()) {
                return Builder_.AddNumericLabel(
                    MapString(label.GetKey()),
                    label.GetNumber()
                );
            }
            TStringId keyId = MapString(label.GetKey());
            if (Policy_.SanitizeThreadNames() && AnyOf(ThreadLabelKeyIds_, [keyId](TStringId id) { return id == keyId; })) {
                return Builder_.AddStringLabel(keyId, SanitizeThreadName(label.GetString()));
            }
            return Builder_.AddStringLabel(keyId, MapString(label.GetString()));
        });
    }

    TStackId MapStack(TStack stack) {
        return Stacks_.TryMap(stack.GetIndex(), [&, this] {
            auto builder = Builder_.AddStack();

            builder.SetTopFrame(MapStackFrame(stack.GetTopFrame()));
            builder.SetStackSegment(MapStackSegment(stack.GetStackSegment()));

            return builder.Finish();
        });
    }

    // Map a stack partially, keeping only specified number of frames.
    // Not cached - partial stacks are unique per stripping point.
    TStackId MapStackPartial(TStack stack, i32 frameCount) {
        auto builder = Builder_.AddStack();
        builder.SetTopFrame(MapStackFrame(stack.GetTopFrame()));

        auto segmentBuilder = Builder_.AddStackSegment();
        TStackSegment segment = stack.GetStackSegment();
        for (TStackFrame frame : segment.GetFrames() | std::views::take(frameCount - 1)) {
            segmentBuilder.AddFrame(MapStackFrame(frame));
        }
        builder.SetStackSegment(segmentBuilder.Finish());

        return builder.Finish();
    }

    TStackSegmentId MapStackSegment(TStackSegment segment) {
        return StackSegments_.TryMap(segment.GetIndex(), [&, this] {
            auto builder = Builder_.AddStackSegment();

            for (auto&& frame : segment.GetFrames()) {
                builder.AddFrame(MapStackFrame(frame));
            }

            return builder.Finish();
        });
    }

    TStackFrameId MapStackFrame(TStackFrame frame) {
        return StackFrames_.TryMap(frame.GetIndex(), [&, this] {
            auto builder = Builder_.AddStackFrame();

            if (!Policy_.MergeBySymbolizedNames()) {
                builder.SetBinary(MapBinary(frame.GetBinary()));
                builder.SetAddress(frame.GetAddress());
            }
            builder.SetInlineChain(MapInlineChain(frame.GetInlineChain()));

            return builder.Finish();
        });
    }

    TBinaryId MapBinary(TBinary binary) {
        return Binaries_.TryMap(binary.GetIndex(), [&, this] {
            auto builder = Builder_.AddBinary();

            builder.SetBuildId(MapString(binary.GetBuildId()));
            builder.SetPath(MapString(binary.GetPath()));
            builder.SetIgnoreBinaryPaths(Policy_.IgnoreBinaryPaths());
            builder.SetHasSkewedAddresses(binary.HasSkewedAddresses());

            return builder.Finish();
        });
    }

    TInlineChainId MapInlineChain(TInlineChain chain) {
        return InlineChains_.TryMap(chain.GetIndex(), [&, this] {
            auto builder = Builder_.AddInlineChain();

            for (TSourceLine line : chain.GetLines()) {
                auto lineBuilder = builder.AddLine();
                if (!Policy_.IgnoreSourceLocations()) {
                    lineBuilder.SetLine(line.GetLine());
                    lineBuilder.SetColumn(line.GetColumn());
                }
                lineBuilder.SetFunction(MapFunction(line.GetFunction()));
                lineBuilder.Finish();
            }

            return builder.Finish();
        });
    }

    TFunctionId MapFunction(TFunction function) {
        return Functions_.TryMap(function.GetIndex(), [&, this] {
            auto builder = Builder_.AddFunction();

            builder.SetName(MapString(function.GetName()));
            builder.SetSystemName(MapString(function.GetSystemName()));
            builder.SetFileName(MapString(function.GetFileName()));
            builder.SetStartLine(function.GetStartLine());

            return builder.Finish();
        });
    }

    TStringId MapString(TProfileString string) {
        return Strings_.TryMap(string.GetIndex(), [&, this] {
            return Builder_.AddString(string.View());
        });
    }

    TStringId MapString(TStringBuf string) {
        return Builder_.AddString(string);
    }

private:
    TProfileBuilder& Builder_;
    const TProfile Profile_;
    const bool IsFirstProfile_;
    const TMergePolicy Policy_;
    ui64 SamplingCounter_;

    TIndexRemapping<TStringId> Strings_;
    TIndexRemapping<TValueTypeId> ValueTypes_;
    TIndexRemapping<TSampleId> Samples_;
    TIndexRemapping<TSampleKeyId> SampleKeys_;
    TIndexRemapping<TStackId> Stacks_;
    TIndexRemapping<TBinaryId> Binaries_;
    TIndexRemapping<TStackSegmentId> StackSegments_;
    TIndexRemapping<TStackFrameId> StackFrames_;
    TIndexRemapping<TInlineChainId> InlineChains_;
    TIndexRemapping<TSourceLineId> SourceLines_;
    TIndexRemapping<TFunctionId> Functions_;
    TIndexRemapping<TLabelGroupId> LabelGroups_;
    TIndexRemapping<TLabelId> Labels_;
    TVector<TStringId> ThreadLabelKeyIds_;
};

////////////////////////////////////////////////////////////////////////////////

class TProfileMerger::TImpl {
public:
    TImpl(NProto::NProfile::Profile* merged)
        : Builder_{merged}
    {}

    NProto::NProfile::Profile* Finish() {
        return std::move(Builder_).Finish();
    }

    void Add(const NProto::NProfile::Profile& proto, const NProto::NProfile::MergeOptions& options) {
        TProfile profile{&proto};
        TSingleProfileMerger{Builder_, options, profile, ProfileCount_++}.Merge();
    }

private:
    TProfileBuilder Builder_;
    ui32 ProfileCount_ = 0;
};

////////////////////////////////////////////////////////////////////////////////

TProfileMerger::TProfileMerger(NProto::NProfile::Profile* merged)
    : Impl_{MakeHolder<TImpl>(merged)}
{}

TProfileMerger::TProfileMerger(TProfileMerger&& rhs) noexcept = default;

TProfileMerger& TProfileMerger::operator=(TProfileMerger&& rhs) noexcept = default;

TProfileMerger::~TProfileMerger() = default;

void TProfileMerger::Add(const NProto::NProfile::Profile& proto, const NProto::NProfile::MergeOptions& options) {
    return Impl_->Add(proto, options);
}

NProto::NProfile::Profile* TProfileMerger::Finish() && {
    return Impl_->Finish();
}

////////////////////////////////////////////////////////////////////////////////

void MergeProfiles(
    TConstArrayRef<NProto::NProfile::Profile> profiles,
    NProto::NProfile::Profile* merged,
    const NProto::NProfile::MergeOptions& options
) {
    TProfileMerger merger{merged};
    for (auto& profile : profiles) {
        merger.Add(profile, options);
    }
    std::move(merger).Finish();
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
