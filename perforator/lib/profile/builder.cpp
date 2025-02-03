#include "builder.h"
#include "library/cpp/iterator/enumerate.h"

#include <absl/container/flat_hash_map.h>

#include <library/cpp/int128/int128.h>

#include <util/memory/pool.h>
#include <util/generic/maybe.h>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

class TStringTableSubview {
public:
    explicit TStringTableSubview(
        NProto::NProfile::StringTable* strtab,
        TStringId index
    )
        : StringTable_{strtab}
        , Index_{index}
    {}

    const char* Data() const {
        return StringTable_->strings().data() + Offset();
    }

    ui32 Offset() const {
        return StringTable_->offset(*Index_);
    }

    ui32 Size() const {
        return StringTable_->length(*Index_);
    }

    const char* begin() const {
        return Data();
    }

    const char* end() const {
        return Data() + Size();
    }

    // The returned TStringBuf can be invalidated on string table modification,
    // so it should not be stored for a long time.
    TStringBuf AsStringBuf() const {
        return {Data(), Size()};
    }

    std::string_view AsStringView() const {
        return AsStringBuf();
    }

    bool operator==(const TStringTableSubview& other) const noexcept {
        return Index_ == other.Index_;
    }

    bool operator!=(const TStringTableSubview& other) const noexcept {
        return !operator==(other);
    }

    struct absl_container_eq {
        using is_transparent = void;

        bool operator()(TStringTableSubview lhs, TStringTableSubview rhs) const {
            return lhs == rhs;
        }

        bool operator()(std::string_view lhs, TStringTableSubview rhs) const {
            return operator()(lhs, rhs.AsStringView());
        }

        bool operator()(TStringTableSubview lhs, std::string_view rhs) const {
            return operator()(lhs.AsStringView(), rhs);
        }

        bool operator()(std::string_view lhs, std::string_view rhs) const {
            return lhs == rhs;
        }
    };

    struct absl_container_hash {
        using is_transparent = void;

        size_t operator()(TStringTableSubview value) const {
            return operator()(value.AsStringView());
        }

        size_t operator()(std::string_view value) const {
            return absl::HashOf(value);
        }
    };

private:
    NProto::NProfile::StringTable* StringTable_;
    TStringId Index_;
};

class TProfileBuilder::TImpl {
public:
    TImpl(TProfileBuilder& owner, NProto::NProfile::Profile& profile)
        : Owner_{owner}
        , Profile_{profile}
    {
        InitializeProfile();
    }

    TMetadataBuilder Metadata() {
        return TMetadataBuilder{Owner_, *Profile_.mutable_metadata()};
    }

    TFeaturesBuilder Features() {
        return TFeaturesBuilder{Owner_, *Profile_.mutable_features()};
    }

    TStringId AddString(TStringBuf string) {
        if (auto it = Strings_.find(string); it != Strings_.end()) {
            return it->second;
        }
        TStringId id = TStringId::FromInternalIndex(static_cast<i32>(Strings_.size()));

        auto& strtab = *Profile_.mutable_strtab();
        Y_ENSURE(*id == strtab.offset_size());
        strtab.add_offset(strtab.strings().size());
        strtab.add_length(string.size());
        strtab.mutable_strings()->append(string);

        TStringTableSubview view{&strtab, id};
        Strings_.emplace(view, id);

        return id;
    }

    TCommentId AddComment(TStringBuf string) {
        return AddComment(AddString(string));
    }

    TCommentId AddComment(TStringId string) {
        i32 id = Profile_.mutable_comments()->comment_size();
        Profile_.mutable_comments()->add_comment(string.GetInternalIndex());
        return TCommentId::FromInternalIndex(id);
    }

    TValueTypeId AddValueType(TStringBuf type, TStringBuf unit) {
        return AddValueType(AddString(type), AddString(unit));
    }

    TValueTypeId AddValueType(TStringId type, TStringId unit) {
        return Fetch(ValueTypes_, TValueTypeInfo{
            .Type = type,
            .Unit = unit,
        });
    }

    TLabelId AddStringLabel(TStringBuf key, TStringBuf value) {
        return AddStringLabel(AddString(key), AddString(value));
    }

    TLabelId AddStringLabel(TStringId key, TStringId value) {
        return AddStringLabel(TStringLabelInfo{
            .Key = key,
            .Value = value,
        });
    }

    TLabelId AddStringLabel(const TStringLabelInfo& info) {
        auto id = Fetch(StringLabels_, info);
        return TLabelId::FromInternalIndex((*id << 1) | 0);
    }

    TLabelId AddNumericLabel(TStringBuf key, i64 value) {
        return AddNumericLabel(AddString(key), value);
    }

    TLabelId AddNumericLabel(TStringId key, i64 value) {
        return AddNumericLabel(TNumberLabelInfo{
            .Key = key,
            .Value = value,
        });
    }

    TLabelId AddNumericLabel(const TNumberLabelInfo& info) {
        TLabelId id = Fetch(NumberLabels_, info);
        return TLabelId::FromInternalIndex((*id << 1) | 1);
    }

    TThreadBuilder AddThread() {
        return {Owner_};
    }

    TThreadId AddThread(const TThreadInfo& info) {
        return FetchHashedLossy(ThreadHashes_, info);
    }

    TBinaryBuilder AddBinary() {
        return {Owner_};
    }

    TBinaryId AddBinary(const TBinaryInfo& info) {
        return Fetch(Binaries_, info);
    }

    TFunctionBuilder AddFunction() {
        return {Owner_};
    }

    TFunctionId AddFunction(const TFunctionInfo& info) {
        return Fetch(Functions_, info);
    }

    TInlineChainBuilder AddInlineChain() {
        return {Owner_};
    }

    TInlineChainId AddInlineChain(const TInlineChainInfo& info) {
        return FetchHashedLossy(InlineChainHashes_, info);
    }

    TStackFrameBuilder AddStackFrame() {
        return {Owner_};
    }

    TStackFrameId AddStackFrame(const TStackFrameInfo& info) {
        return Fetch(StackFrames_, info);
    }

    TStackBuilder AddStack() {
        return {Owner_};
    }

    TStackId AddStack(const TStackInfo& info) {
        return FetchHashedLossy(StackHashes_, info);
    }

    TSampleKeyBuilder AddSampleKey() {
        return {Owner_};
    }

    TSampleKeyId AddSampleKey(const TSampleKeyInfo& info) {
        TSampleKeyId id = FetchHashedLossy(SampleKeyHashes_, info);
        if (ui32 idx = id.GetInternalIndex(); idx >= SampleByKeys_.size()) {
            SampleByKeys_.resize(1 + id.GetInternalIndex() * 2);
        }
        return id;
    }

    TSampleBuilder AddSample() {
        return {Owner_};
    }

    TSampleId AddSample(const TSampleInfo& info) {
        TSampleId id = PrepareSample(info);
        FillSampleValues(id, info);
        return id;
    }

    void Finish() {
        for (auto [i, sum] : Enumerate(ValuesSum_)) {
            auto& protosum = *Profile_.mutable_samples()->mutable_values(i)->mutable_value_sum();
            protosum.set_lo(GetLow(sum));
            protosum.set_hi(GetHigh(sum));
        }
    }

private:
    void InitializeProfile() {
        AddString("");
        FetchHashedLossy(ThreadHashes_, TThreadInfo{});
        Fetch(Binaries_, TBinaryInfo{});
        Fetch(Functions_, TFunctionInfo{});
        FetchHashedLossy(InlineChainHashes_, TInlineChainInfo{});
        FetchHashedLossy(StackHashes_, TStackInfo{});
        Fetch(StackFrames_, TStackFrameInfo{});
    }

    TSampleId PrepareSample(const TSampleInfo& sample) {
        TSampleId id = TSampleId::FromInternalIndex(Profile_.samples().key_size());

        // We should not merge timestamped samples.
        if (!sample.Timestamp) {
            TMaybe<TSampleId>& prev = SampleByKeys_.at(*sample.Key);
            if (prev) {
                id = *prev;
           } else {
                prev = id;
           }
        }

        // If this is a new sample, let's register it.
        if (*id >= Profile_.samples().key_size()) {
            Y_ENSURE(*id == Profile_.samples().key_size());
            Profile_.mutable_samples()->add_key(*sample.Key);
            if (auto&& ts = sample.Timestamp) {
                FillSampleTimestamp(id, *ts);
            }

            AllocateSampleValues(id);
        }

        return id;
    }

    void FillSampleTimestamp(TSampleId id, TSampleTimestamp ts) {
        auto&& timestamps = *Profile_.mutable_samples()->mutable_timestamps();
        if (!timestamps.has_start_timestamp()) {
            Y_ENSURE(*id == 0, "Cannot build profile with a mix of timestamped & non-timestamped samples");
            timestamps.mutable_start_timestamp()->set_seconds(ts.Seconds);
            timestamps.mutable_start_timestamp()->set_nanos(ts.NanoSeconds);
        }

        constexpr i64 nanoSecondsInSecond = 1'000'000'000ll;
        const google::protobuf::Timestamp& start = timestamps.start_timestamp();

        i64 deltaSeconds = ts.Seconds - start.seconds();
        i64 deltaNanoSeconds = static_cast<i64>(ts.NanoSeconds) - start.nanos();
        while (deltaNanoSeconds < 0) {
            --deltaSeconds;
            deltaNanoSeconds += nanoSecondsInSecond;
        }

        timestamps.add_delta_nanoseconds(nanoSecondsInSecond * deltaSeconds + deltaNanoSeconds);
    }

    void AllocateSampleValues(TSampleId id) {
        for (NProto::NProfile::SampleValues& values : *Profile_.mutable_samples()->mutable_values()) {
            Y_ENSURE(values.value_size() == *id);
            values.add_value(0);
        }
    }

    void FillSampleValues(TSampleId sampleId, const TSampleInfo& sample) {
        for (auto [valueId, delta] : sample.Values) {
            NProto::NProfile::SampleValues& values = *Profile_.mutable_samples()->mutable_values(*valueId);
            values.mutable_value()->at(*sampleId) += delta;

            Y_ASSERT(static_cast<ui32>(*valueId) < ValuesSum_.size());
            ValuesSum_[*valueId] += delta;
        }
    }

    template <typename Map, typename Key, typename Entity, typename Index = typename Map::mapped_type>
    Index FetchImpl(Map& map, const Key& key, const Entity& entity) {
        Y_ENSURE(map.size() <= Max<ui32>());

        auto index = Index::FromInternalIndex(static_cast<ui32>(map.size()));
        auto [it, ok] = map.try_emplace(key, index);
        if (ok) {
            FillEntityAt(entity, index);
        }
        return it->second;
    }

    template <typename Map, typename Entity, typename Index = typename Map::mapped_type>
    Index Fetch(Map& map, const Entity& entity) {
        return FetchImpl(map, entity, entity);
    }

    template <typename Map, typename Key, typename Index = typename Map::mapped_type>
    Index FetchHashedLossy(Map& map, Key key) {
        return FetchImpl(map, absl::HashOf(key) ^ 0xdeadbeefdeadbeefull, key);
    }

private:
    void FillEntityAt(TStringBuf str, TStringId id) {
        auto& strtab = *Profile_.mutable_strtab();
        Y_ENSURE(strtab.length_size() == *id);
        strtab.mutable_strings()->AppendNoAlias(str);
    }

    void FillEntityAt(const TValueTypeInfo& info, TValueTypeId id) {
        auto& values = *Profile_.mutable_samples()->mutable_values();
        for (auto&& sampleValues : values) {
            Y_ENSURE(sampleValues.value_size() == 0, "Trying to add value type for non-empty profile");
        }

        Y_ENSURE(values.size() == *id);
        auto* sampleValues = values.Add();
        sampleValues->mutable_type()->set_type(info.Type.GetInternalIndex());
        sampleValues->mutable_type()->set_unit(info.Unit.GetInternalIndex());

        Y_ENSURE(ValuesSum_.size() == static_cast<size_t>(*id));
        ValuesSum_.emplace_back(0);
    }

    void FillEntityAt(const TStringLabelInfo& info, TLabelId id) {
        auto& labels = *Profile_.mutable_labels()->mutable_strings();
        CheckedAddAt(labels.mutable_key(), id, *info.Key);
        CheckedAddAt(labels.mutable_value(), id, *info.Value);
    }

    void FillEntityAt(const TNumberLabelInfo& info, TLabelId id) {
        auto& labels = *Profile_.mutable_labels()->mutable_numbers();
        CheckedAddAt(labels.mutable_key(), id, *info.Key);
        CheckedAddAt(labels.mutable_value(), id, info.Value);
    }

    void FillEntityAt(const TThreadInfo& info, TThreadId id) {
        auto& threads = *Profile_.mutable_threads();
        CheckedAddAt(threads.mutable_thread_id(), id, info.ThreadId);
        CheckedAddAt(threads.mutable_process_id(), id, info.ProcessId);
        CheckedAddAt(threads.mutable_thread_name(), id, *info.ThreadNameIdx);
        CheckedAddAt(threads.mutable_process_name(), id, *info.ProcessNameIdx);
        CheckedAddAt(threads.mutable_container_offset(), id, threads.container_names_size());
        for (TStringId container : info.ContainerIdx) {
            threads.add_container_names(*container);
        }
    }

    void FillEntityAt(const TBinaryInfo& info, TBinaryId id) {
        auto& binaries = *Profile_.mutable_binaries();
        CheckedAddAt(binaries.mutable_build_id(), id, *info.BuildId);
        CheckedAddAt(binaries.mutable_path(), id, *info.Path);
    }

    void FillEntityAt(const TFunctionInfo& info, TFunctionId id) {
        auto& binaries = *Profile_.mutable_functions();
        CheckedAddAt(binaries.mutable_name(), id, *info.Name);
        CheckedAddAt(binaries.mutable_system_name(), id, *info.SystemName);
        CheckedAddAt(binaries.mutable_filename(), id, *info.FileName);
        CheckedAddAt(binaries.mutable_start_line(), id, info.StartLine);
    }

    void FillEntityAt(const TInlineChainInfo& info, TInlineChainId id) {
        auto& chains = *Profile_.mutable_inline_chains();
        CheckedAddAt(chains.mutable_offset(), id, chains.function_id_size());
        for (const TSourceLineInfo& line : info.Lines) {
            chains.add_function_id(*line.Function);
            chains.add_line(line.Line);
            chains.add_column(line.Column);
        }
    }

    void FillEntityAt(const TStackFrameInfo& info, TStackFrameId id) {
        auto& frames = *Profile_.mutable_stack_frames();
        CheckedAddAt(frames.mutable_binary_id(), id, *info.Binary);
        CheckedAddAt(frames.mutable_binary_offset(), id, info.BinaryOffset);
        CheckedAddAt(frames.mutable_inline_chain_id(), id, *info.InlineChain);
    }

    void FillEntityAt(const TStackInfo& info, TStackId id) {
        auto& stacks = *Profile_.mutable_stacks();
        CheckedAddAt(stacks.mutable_offset(), id, stacks.frame_id_size());
        for (const TStackFrameId& frame : info.Stack) {
            stacks.add_frame_id(*frame);
        }
    }

    void FillEntityAt(const TSampleKeyInfo& info, TSampleKeyId id) {
        auto& keys = *Profile_.mutable_sample_keys();
        CheckedAddAt(keys.mutable_threads()->mutable_thread_id(), id, *info.Thread);
        CheckedAddAt(keys.mutable_stacks()->mutable_kernel_stack_id(), id, *info.KernelStack);
        CheckedAddAt(keys.mutable_stacks()->mutable_user_stack_id(), id, *info.UserStack);
        CheckedAddAt(keys.mutable_labels()->mutable_first_label_id(), id, keys.labels().packed_label_id_size());
        for (auto&& label : info.Labels) {
            keys.mutable_labels()->add_packed_label_id(*label);
        }
    }

private:
    template <typename Arr, typename Index, typename Value>
    void CheckedAddAt(Arr* array, Index index, const Value& value) {
        Y_ENSURE(array->size() == *index);
        array->Add(value);
    }

private:
    TProfileBuilder& Owner_;
    NProto::NProfile::Profile& Profile_;

    TMemoryPool Pool_{4096};
    absl::flat_hash_map<TStringTableSubview, TStringId> Strings_;
    absl::flat_hash_map<TValueTypeInfo, TValueTypeId> ValueTypes_;
    absl::flat_hash_map<TStringLabelInfo, TLabelId> StringLabels_;
    absl::flat_hash_map<TNumberLabelInfo, TLabelId> NumberLabels_;
    absl::flat_hash_map<ui64, TThreadId> ThreadHashes_;
    absl::flat_hash_map<TBinaryInfo, TBinaryId> Binaries_;
    absl::flat_hash_map<TFunctionInfo, TFunctionId> Functions_;
    absl::flat_hash_map<ui64, TInlineChainId> InlineChainHashes_;
    absl::flat_hash_map<TStackFrameInfo, TStackFrameId> StackFrames_;
    absl::flat_hash_map<ui64, TStackId> StackHashes_;
    absl::flat_hash_map<ui64, TSampleKeyId> SampleKeyHashes_;
    TVector<TMaybe<TSampleId>> SampleByKeys_;
    TVector<ui128> ValuesSum_;
};

////////////////////////////////////////////////////////////////////////////////

TProfileBuilder::TProfileBuilder(NProto::NProfile::Profile* profile) {
    Y_ENSURE(profile);
    Impl_ = MakeHolder<TImpl>(*this, *profile);
}

TProfileBuilder::~TProfileBuilder() = default;

TProfileBuilder::TMetadataBuilder TProfileBuilder::Metadata() {
    return Impl_->Metadata();
}

TProfileBuilder::TFeaturesBuilder TProfileBuilder::Features() {
    return Impl_->Features();
}

////////////////////////////////////////////////////////////////////////////////

TStringId TProfileBuilder::AddString(TStringBuf string) {
    return Impl_->AddString(string);
}

////////////////////////////////////////////////////////////////////////////////

TCommentId TProfileBuilder::AddComment(TStringBuf string) {
    return Impl_->AddComment(string);
}

TCommentId TProfileBuilder::AddComment(TStringId string) {
    return Impl_->AddComment(string);
}

////////////////////////////////////////////////////////////////////////////////

TValueTypeId TProfileBuilder::AddValueType(TStringBuf type, TStringBuf unit) {
    return Impl_->AddValueType(type, unit);
}

TValueTypeId TProfileBuilder::AddValueType(TStringId type, TStringId unit) {
    return Impl_->AddValueType(type, unit);
}

////////////////////////////////////////////////////////////////////////////////

TLabelId TProfileBuilder::AddStringLabel(TStringBuf key, TStringBuf value) {
    return Impl_->AddStringLabel(key, value);
}

TLabelId TProfileBuilder::AddStringLabel(TStringId key, TStringId value) {
    return Impl_->AddStringLabel(key, value);
}

TLabelId TProfileBuilder::AddNumericLabel(TStringBuf key, i64 value) {
    return Impl_->AddNumericLabel(key, value);
}

TLabelId TProfileBuilder::AddNumericLabel(TStringId key, i64 value) {
    return Impl_->AddNumericLabel(key, value);
}

////////////////////////////////////////////////////////////////////////////////

TProfileBuilder::TThreadBuilder TProfileBuilder::AddThread() {
    return Impl_->AddThread();
}

TThreadId TProfileBuilder::AddThread(const TThreadInfo& info) {
    return Impl_->AddThread(info);
}

////////////////////////////////////////////////////////////////////////////////

TProfileBuilder::TBinaryBuilder TProfileBuilder::AddBinary() {
    return Impl_->AddBinary();
}

TBinaryId TProfileBuilder::AddBinary(const TBinaryInfo& info) {
    return Impl_->AddBinary(info);
}

////////////////////////////////////////////////////////////////////////////////

TProfileBuilder::TFunctionBuilder TProfileBuilder::AddFunction() {
    return Impl_->AddFunction();
}

TFunctionId TProfileBuilder::AddFunction(const TFunctionInfo& info) {
    return Impl_->AddFunction(info);
}

////////////////////////////////////////////////////////////////////////////////

TProfileBuilder::TInlineChainBuilder TProfileBuilder::AddInlineChain() {
    return Impl_->AddInlineChain();
}

TInlineChainId TProfileBuilder::AddInlineChain(const TInlineChainInfo& info) {
    return Impl_->AddInlineChain(info);
}

////////////////////////////////////////////////////////////////////////////////

TProfileBuilder::TStackFrameBuilder TProfileBuilder::AddStackFrame() {
    return Impl_->AddStackFrame();
}

TStackFrameId TProfileBuilder::AddStackFrame(const TStackFrameInfo& info) {
    return Impl_->AddStackFrame(info);
}

////////////////////////////////////////////////////////////////////////////////

TProfileBuilder::TStackBuilder TProfileBuilder::AddStack() {
    return Impl_->AddStack();
}

TStackId TProfileBuilder::AddStack(const TStackInfo& info) {
    return Impl_->AddStack(info);
}

////////////////////////////////////////////////////////////////////////////////

TProfileBuilder::TSampleKeyBuilder TProfileBuilder::AddSampleKey() {
    return Impl_->AddSampleKey();
}

TSampleKeyId TProfileBuilder::AddSampleKey(const TSampleKeyInfo& info) {
    return Impl_->AddSampleKey(info);
}

////////////////////////////////////////////////////////////////////////////////

TProfileBuilder::TSampleBuilder TProfileBuilder::AddSample() {
    return Impl_->AddSample();
}

TSampleId TProfileBuilder::AddSample(const TSampleInfo& info) {
    return Impl_->AddSample(info);
}

////////////////////////////////////////////////////////////////////////////////

void TProfileBuilder::Finish() {
    return Impl_->Finish();
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
