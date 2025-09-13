#include "builder.h"
#include "entity_index.h"
#include "merge.h"
#include "profile.h"

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>
#include <library/cpp/containers/absl_flat_hash/flat_hash_set.h>

#include <util/system/mutex.h>

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

    bool IgnoreThreadIds() const {
        return Options_.ignore_thread_ids();
    }

    bool IgnoreProcessIds() const {
        return Options_.ignore_process_ids();
    }

    bool IgnoreThreadNames() const {
        return Options_.ignore_thread_names();
    }

    bool IgnoreProcessNames() const {
        return Options_.ignore_process_names();
    }

    bool MergeBinaries() const {
        return Options_.merge_by_symbolized_names();
    }

    bool IgnoreBinaryPaths() const {
        return Options_.ignore_binary_paths();
    }

    bool MergeTimestamps() const {
        return Options_.ignore_timestamps();
    }

    bool MergeSourceLocations() const {
        return Options_.ignore_source_locations();
    }

    bool MergeBinaryAddresses() const {
        return Options_.ignore_binary_addresses();
    }

    bool CleanupThreadNames() const {
        return Options_.cleanup_thread_names();
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
            && SampleHasAllOfRequiredLabels(sample)
        ;
    }

    bool AllowLabel(TLabel label) const {
        return !DroppedLabelKeys_.contains(label.GetKey().GetIndex());
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
        if (Options_.label_filter().skipped_key_prefixes().empty()) {
            return;
        }

        for (TLabel label : Profile_.Labels()) {
            for (TStringBuf prefix : Options_.label_filter().skipped_key_prefixes()) {
                if (label.GetKey().View().StartsWith(prefix)) {
                    DroppedLabelKeys_.insert(label.GetKey().GetIndex());
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
    absl::flat_hash_set<TStringId> DroppedLabelKeys_;
};

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
        , Strings_{*Profile_.Strings().GetPastTheEndIndex()}
        , ValueTypes_{*Profile_.ValueTypes().GetPastTheEndIndex()}
        , Samples_{*Profile_.Samples().GetPastTheEndIndex()}
        , SampleKeys_{*Profile_.SampleKeys().GetPastTheEndIndex()}
        , Stacks_{*Profile_.Stacks().GetPastTheEndIndex()}
        , Binaries_{*Profile_.Binaries().GetPastTheEndIndex()}
        , StackSegments_{*Profile_.StackSegments().GetPastTheEndIndex()}
        , StackFrames_{*Profile_.StackFrames().GetPastTheEndIndex()}
        , InlineChains_{*Profile_.InlineChains().GetPastTheEndIndex()}
        , SourceLines_{*Profile_.SourceLines().GetPastTheEndIndex()}
        , Functions_{*Profile_.Functions().GetPastTheEndIndex()}
        , Threads_{*Profile_.Threads().GetPastTheEndIndex()}
        , Labels_{*Profile_.Labels().GetPastTheEndIndex()}
    {}

    void Merge() {
        MergeFeatures();
        MergeMetadata();
        MergeSamples();
    }

private:
    void MergeFeatures() {
        auto&& prev = Builder_.Features().GetProto();
        auto&& curr = Profile_.GetFeatures();

        if (IsFirstProfile_) {
            prev.set_has_skewed_binary_offsets(curr.has_skewed_binary_offsets());
        } else {
            Y_ENSURE(prev.has_skewed_binary_offsets() == curr.has_skewed_binary_offsets());
        }
    }

    void MergeMetadata() {
        auto&& prev = Builder_.Metadata().GetProto();
        auto&& curr = Profile_.GetMetadata();

        TStringRef str = Profile_.Strings().Get(curr.default_sample_type());
        TStringId defaultSampleType = MapString(str);

        if (IsFirstProfile_) {
            prev.set_default_sample_type(*defaultSampleType);
        } else {
            Y_ENSURE(prev.default_sample_type() == (ui32)*defaultSampleType);
        }
    }

    void MergeSamples() {
        for (TSample sample : Profile_.Samples()) {
            if (Policy_.AllowSample(sample)) {
                MergeSample(sample);
            }
        }
    }

    void MergeSample(TSample sample) {
        auto builder = Builder_.AddSample();

        if (auto ts = sample.GetProtoTimestamp(); ts && !Policy_.MergeTimestamps()) {
            builder.SetTimestamp(ts->seconds(), ts->nanos());
        }

        builder.SetSampleKey(MapSampleKey(sample.GetKey()));

        for (i32 i = 0; i < sample.GetValueCount(); ++i) {
            builder.AddValue(MapValueType(sample.GetValueType(i)), sample.GetValue(i));
        }

        builder.Finish();
    }

    TValueTypeId MapValueType(TValueType type) {
        // TODO(sskvor): If different profiles have different value type sets,
        // the process of merging should fail inside TProfileBuilder on the first
        // call to "AddValueType" that follows calls to "AddSample".
        return ValueTypes_.TryMap(type.GetIndex(), [&, this] {
            return Builder_.AddValueType(
                MapString(type.GetType()),
                MapString(type.GetUnit())
            );
        });
    }

    TSampleKeyId MapSampleKey(TSampleKey key) {
        return SampleKeys_.TryMap(key.GetIndex(), [&key, this] {
            auto builder = Builder_.AddSampleKey();

            builder.SetThread(MapThread(key.GetThread()));

            for (TStack stack : key.GetStacks()) {
                builder.AddStack(MapStack(stack));
            }

            for (TLabel label : key.GetLabels()) {
                if (Policy_.AllowLabel(label)) {
                    builder.AddLabel(MapLabel(label));
                }
            }

            return builder.Finish();
        });
    }

    TThreadId MapThread(TThread thread) {
        return Threads_.TryMap(thread.GetIndex(), [&thread, this] {
            auto builder = Builder_.AddThread();

            if (!Policy_.IgnoreThreadIds()) {
                builder.SetThreadId(thread.GetThreadId());
            }
            if (!Policy_.IgnoreProcessIds()) {
                builder.SetProcessId(thread.GetProcessId());
            }

            if (!Policy_.IgnoreThreadNames()) {
                if (Policy_.CleanupThreadNames()) {
                    TStringId sanitized = SanitizeThreadName(thread.GetThreadName());
                    builder.SetThreadName(sanitized);
                } else {
                    builder.SetThreadName(MapString(thread.GetThreadName()));
                }
            }

            if (!Policy_.IgnoreProcessNames()) {
                builder.SetProcessName(MapString(thread.GetProcessName()));
            }

            for (TStringRef container : thread.GetContainers()) {
                builder.AddContainerName(MapString(container));
            }

            return builder.Finish();
        });
    }

    TStringId SanitizeThreadName(TStringRef name) {
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
            } else {
                return Builder_.AddStringLabel(
                    MapString(label.GetKey()),
                    MapString(label.GetString())
                );
            }
        });
    }

    TStackId MapStack(TStack stack) {
        return Stacks_.TryMap(stack.GetIndex(), [&, this] {
            auto builder = Builder_.AddStack();

            builder.SetKind(stack.GetKind());
            builder.SetRuntimeName(MapString(stack.GetRuntimeName()));
            builder.SetTopFrame(MapStackFrame(stack.GetTopFrame()));
            builder.SetStackSegment(MapStackSegment(stack.GetStackSegment()));

            return builder.Finish();
        });
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

            if (!Policy_.MergeBinaries()) {
                builder.SetBinary(MapBinary(frame.GetBinary()));
                builder.SetBinaryOffset(frame.GetBinaryOffset());
            }
            builder.SetInlineChain(MapInlineChain(frame.GetInlineChain()));

            return builder.Finish();
        });
    }

    TBinaryId MapBinary(TBinary binary) {
        return Binaries_.TryMap(binary.GetIndex(), [&, this] {
            auto builder = Builder_.AddBinary();

            builder.SetBuildId(MapString(binary.GetBuildId()));
            if (!Policy_.IgnoreBinaryPaths()) {
                builder.SetPath(MapString(binary.GetPath()));
            }

            return builder.Finish();
        });
    }

    TInlineChainId MapInlineChain(TInlineChain chain) {
        return InlineChains_.TryMap(chain.GetIndex(), [&, this] {
            auto builder = Builder_.AddInlineChain();

            for (TSourceLine line : chain.GetLines()) {
                auto lineBuilder = builder.AddLine();
                if (!Policy_.MergeSourceLocations()) {
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

    TStringId MapString(TStringRef string) {
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
    TIndexRemapping<TThreadId> Threads_;
    TIndexRemapping<TLabelId> Labels_;
};

////////////////////////////////////////////////////////////////////////////////

class TProfileMerger::TImpl {
public:
    TImpl(NProto::NProfile::Profile* merged, const NProto::NProfile::MergeOptions& options)
        : Options_{options}
        , Builder_{merged}
    {}

    NProto::NProfile::Profile* Finish() {
        return std::move(Builder_).Finish();
    }

    void Add(const NProto::NProfile::Profile& proto) {
        TProfile profile{&proto};
        TSingleProfileMerger{Builder_, Options_, profile, ProfileCount_++}.Merge();
    }

private:
    const NProto::NProfile::MergeOptions& Options_;
    TProfileBuilder Builder_;
    ui32 ProfileCount_ = 0;
};

////////////////////////////////////////////////////////////////////////////////

TProfileMerger::TProfileMerger(NProto::NProfile::Profile* merged, const NProto::NProfile::MergeOptions& options)
    : Impl_{MakeHolder<TImpl>(merged, options)}
{}

TProfileMerger::TProfileMerger(TProfileMerger&& rhs) noexcept = default;

TProfileMerger& TProfileMerger::operator=(TProfileMerger&& rhs) noexcept = default;

TProfileMerger::~TProfileMerger() = default;

void TProfileMerger::Add(const NProto::NProfile::Profile& proto) {
    return Impl_->Add(proto);
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
    TProfileMerger merger{merged, options};
    for (auto& profile : profiles) {
        merger.Add(profile);
    }
    std::move(merger).Finish();
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
