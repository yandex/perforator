#pragma once

#include "entity_index.h"

#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/json/json_writer.h>

#include <util/datetime/base.h>
#include <util/generic/function_ref.h>
#include <util/generic/iterator.h>
#include <util/generic/maybe.h>
#include <util/generic/yexception.h>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

template <typename Array>
static std::pair<int, int> GetOffsetRange(Array&& offsets, Array&& values, int rangeId) {
    if (rangeId >= offsets.size()) {
        return {0, 0};
    }

    int begin = offsets.at(rangeId);
    int end = 0;
    if (int nextRangeId = rangeId + 1; nextRangeId < offsets.size()) {
        end = offsets.at(nextRangeId);
    } else {
        end = values.size();
    }
    return {begin, end};
}

template <typename Offsets, typename Array, typename F>
static void IterateRange(Offsets&& offsets, Array&& values, int rangeId, F&& func) {
    auto [begin, end] = GetOffsetRange(offsets, values, rangeId);
    for (; begin != end; ++begin) {
        func(values.at(begin));
    }
}

////////////////////////////////////////////////////////////////////////////////

// Non-owning accessor to a profile string table.
// Must not outlive the original profile.
class TStringTable {
public:
    explicit TStringTable(const NProto::NProfile::StringTable* strtab)
        : StringTable_{strtab}
    {
        Y_ENSURE(StringTable_ != nullptr);
        Y_ENSURE(StringTable_->offset_size() == StringTable_->length_size());
    }

    TStringBuf Get(int index) const {
        Y_ENSURE(index < StringTable_->offset_size());

        ui32 offset = StringTable_->offset(index);
        ui32 length = StringTable_->length(index);
        TStringBuf strings = StringTable_->strings();

        Y_ENSURE(offset + length <= strings.size());
        return strings.SubString(offset, length);
    }

private:
    const NProto::NProfile::StringTable* StringTable_;
};


////////////////////////////////////////////////////////////////////////////////

template <CStrongIndex Index>
struct TEntityTraits;

template <CStrongIndex Index>
struct TCommonDenseIndexTraits {
    using TIndex = Index;

    static TIndex GetPastTheEndIndex(const NProto::NProfile::Profile& profile) {
        const i32 count = TEntityTraits<Index>::GetEntityCount(profile);
        return TIndex::FromInternalIndex(count);
    }

    static bool IsValidIndex(const NProto::NProfile::Profile& profile, Index index) {
        const TIndex last = GetPastTheEndIndex(profile);
        return index.GetInternalIndex() < last.GetInternalIndex();
    }

    static TIndex GetNextIndex(const NProto::NProfile::Profile&, TIndex index) {
        return TIndex::FromInternalIndex(index.GetInternalIndex() + 1);
    }
};

template <>
struct TEntityTraits<TStringId> : TCommonDenseIndexTraits<TStringId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.strtab().length_size();
    }
};

template <>
struct TEntityTraits<TCommentId> : TCommonDenseIndexTraits<TCommentId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.comments().comment_size();
    }
};

template <>
struct TEntityTraits<TValueTypeId> : TCommonDenseIndexTraits<TValueTypeId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.samples().values_size();
    }
};

template <>
struct TEntityTraits<TFunctionId> : TCommonDenseIndexTraits<TFunctionId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.functions().name_size();
    }
};

template <>
struct TEntityTraits<TBinaryId> : TCommonDenseIndexTraits<TBinaryId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.binaries().build_id_size();
    }
};

template <>
struct TEntityTraits<TSourceLineId> : TCommonDenseIndexTraits<TSourceLineId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.inline_chains().function_id_size();
    }
};

template <>
struct TEntityTraits<TInlineChainId> : TCommonDenseIndexTraits<TInlineChainId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.inline_chains().offset_size();
    }
};

template <>
struct TEntityTraits<TStackFrameId> : TCommonDenseIndexTraits<TStackFrameId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.stack_frames().binary_id_size();
    }
};

template <>
struct TEntityTraits<TStackId> : TCommonDenseIndexTraits<TStackId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.stacks().offset_size();
    }
};

template <>
struct TEntityTraits<TThreadId> : TCommonDenseIndexTraits<TThreadId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.threads().thread_id_size();
    }
};

template <>
struct TEntityTraits<TSampleKeyId> : TCommonDenseIndexTraits<TSampleKeyId> {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.sample_keys().threads().thread_id_size();
    }
};

template <>
struct TEntityTraits<TSampleId> : TCommonDenseIndexTraits<TSampleId>  {
    static i32 GetEntityCount(const NProto::NProfile::Profile& profile) {
        return profile.samples().key_size();
    }
};

template <>
struct TEntityTraits<TLabelId> {
    using TIndex = TLabelId;

    static TIndex GetPastTheEndIndex(const NProto::NProfile::Profile& profile) {
        return TIndex::FromInternalIndex(1 + Max(
            (profile.labels().strings().key_size() << 1) | 0,
            (profile.labels().numbers().key_size() << 1) | 1
        ));
    }

    static bool IsValidIndex(const NProto::NProfile::Profile& profile, TIndex index) {
        auto unpacked = GetUnpackedIndex(index);

        switch (GetTypeTag(index)) {
        case 0:
            return unpacked < profile.labels().strings().key_size();
        case 1:
            return unpacked < profile.labels().numbers().key_size();
        default:
            Y_ENSURE(false, "Unsupported label type tag");
        }
    }

    static TIndex GetNextIndex(const NProto::NProfile::Profile& profile, TIndex index) {
        const TIndex last = GetPastTheEndIndex(profile);

        TIndex next = index;
        do {
            next = TIndex::FromInternalIndex(next.GetInternalIndex() + 1);
        } while (!IsValidIndex(profile, next) && next < last);

        return next;
    }

private:
    static i32 GetTypeTag(TIndex index) {
        return index.GetInternalIndex() & 1;
    }

    static i32 GetUnpackedIndex(TIndex index) {
        return index.GetInternalIndex() >> 1;
    }
};

////////////////////////////////////////////////////////////////////////////////

template <CStrongIndex Index>
class TIndexedEntityReader {
public:
    using TTraits = TEntityTraits<Index>;
    using TIndex = Index;
    using TBase = TIndexedEntityReader<Index>;

    TIndexedEntityReader(const NProto::NProfile::Profile* profile, ui32 id)
        : TIndexedEntityReader{profile, TIndex::FromInternalIndex(id)}
    {}

    TIndexedEntityReader(const NProto::NProfile::Profile* profile, Index id)
        : Profile_{profile}
        , Index_{id}
    {
        Y_ENSURE(TTraits::IsValidIndex(*profile, id));
    }

    Index GetIndex() const {
        return Index_;
    }

protected:
    const NProto::NProfile::Profile* Profile_ = nullptr;
    Index Index_;
};

template <typename TEntity>
class TEntityArray {
    using TIndex = typename TEntity::TIndex;
    using TTraits = typename TEntity::TTraits;

public:
    class TIterator;

public:
    TEntityArray(const NProto::NProfile::Profile* profile)
        : Profile_{profile}
    {}

    TIndex GetPastTheEndIndex() const {
        return TTraits::GetPastTheEndIndex(*Profile_);
    }

    size_t GetApproxSize() const {
        return *GetPastTheEndIndex();
    }

    TEntity Get(TIndex index) const {
        return TEntity{Profile_, index};
    }

    TEntity Get(i32 index) const {
        return Get(TIndex::FromInternalIndex(index));
    }

    TIterator begin() const {
        return TIterator{TIndex::Zero(), this};
    }

    TIterator end() const {
        return TIterator{};
    }

public:
    class TIterator {
    public:
        TIterator() = default;

        TIterator(TIndex index, const TEntityArray* array)
            : Index_{index}
            , Array_{array}
        {}

        bool IsExhausted() const {
            if (!Array_) {
                return true;
            }
            return !TTraits::IsValidIndex(*Array_->Profile_, Index_);
        }

        bool operator==(const TIterator& other) const noexcept {
            if (IsExhausted() || other.IsExhausted()) {
                return IsExhausted() == other.IsExhausted();
            }

            return Index_ == other.Index_;
        }

        bool operator!=(const TIterator& other) const noexcept {
            return !operator==(other);
        }

        TIterator operator++(int) {
            TIterator copy{*this};
            ++*this;
            return copy;
        }

        TIterator& operator++() {
            Index_ = TTraits::GetNextIndex(*Array_->Profile_, Index_);
            return *this;
        }

        TEntity operator*() const {
            return Array_->Get(Index_);
        }

        TEntity operator->() const {
            return operator*();
        }

    private:
        TIndex Index_ = TIndex::Invalid();
        const TEntityArray* Array_ = nullptr;
    };

private:
    const NProto::NProfile::Profile* Profile_ = nullptr;
};

////////////////////////////////////////////////////////////////////////////////

class TStringRef : public TIndexedEntityReader<TStringId> {
public:
    using TBase::TBase;

    TStringBuf View() const {
        return TStringTable{&Profile_->strtab()}.Get(*Index_);
    }

    explicit operator bool() const {
        return 0 != *Index_;
    }
};

class TFunction : public TIndexedEntityReader<TFunctionId> {
public:
    using TBase::TBase;

    TStringRef GetName() const {
        return {Profile_, Profile_->functions().name(*Index_)};
    }

    TStringRef GetSystemName() const {
        return {Profile_, Profile_->functions().system_name(*Index_)};
    }

    TStringRef GetFileName() const {
        return {Profile_, Profile_->functions().filename(*Index_)};
    }

    ui32 GetStartLine() const {
        return Profile_->functions().start_line(*Index_);
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "function");
        writer.Write("id", *GetIndex());

        writer.Write("name", GetName().View());
        writer.Write("system_name", GetSystemName().View());
        writer.Write("file_name", GetFileName().View());
        writer.Write("start_line", GetStartLine());

        writer.CloseMap();
    }
};

class TSourceLine : public TIndexedEntityReader<TSourceLineId> {
public:
    using TBase::TBase;

    TFunction GetFunction() const {
        i32 functionId = Profile_->inline_chains().function_id(*Index_);
        return TFunction{Profile_, TFunctionId::FromInternalIndex(functionId)};
    }

    ui32 GetLine() const {
        return Profile_->inline_chains().line(*Index_);
    }

    ui32 GetColumn() const {
        return Profile_->inline_chains().column(*Index_);
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "source_line");
        writer.Write("id", *GetIndex());

        writer.WriteKey("function");
        GetFunction().DumpJson(writer);

        writer.Write("line", GetLine());
        writer.Write("column", GetColumn());

        writer.CloseMap();
    }
};

class TInlineChain : public TIndexedEntityReader<TInlineChainId> {
public:
    using TBase::TBase;

    i32 GetLineCount() const {
        auto [from, to] = GetOffsetRange(
            Profile_->inline_chains().offset(),
            Profile_->inline_chains().function_id(),
            *Index_
        );

        return to - from;
    }

    TSourceLine GetLine(i32 id) const {
        i32 offset = Profile_->inline_chains().offset(*Index_);
        return TSourceLine{Profile_, TSourceLineId::FromInternalIndex(offset + id)};
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "inline_chain");
        writer.Write("id", *GetIndex());

        writer.WriteKey("lines");
        writer.OpenArray();
        for (i32 i = 0; i < GetLineCount(); ++i) {
            GetLine(i).DumpJson(writer);
        }
        writer.CloseArray();

        writer.CloseMap();
    }
};

class TBinary : public TIndexedEntityReader<TBinaryId> {
public:
    using TBase::TBase;

    TStringRef GetBuildId() const {
        ui32 id = Profile_->binaries().build_id(*Index_);
        return {Profile_, id};
    }

    TStringRef GetPath() const {
        ui32 id = Profile_->binaries().path(*Index_);
        return {Profile_, id};
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "binary");
        writer.Write("id", *GetIndex());

        writer.Write("build_id", GetBuildId().View());
        writer.Write("path", GetPath().View());

        writer.CloseMap();
    }
};

class TStackFrame : public TIndexedEntityReader<TStackFrameId> {
public:
    using TBase::TBase;

    TBinary GetBinary() const {
        i32 index = Profile_->stack_frames().binary_id(*Index_);
        return TBinary{Profile_, TBinaryId::FromInternalIndex(index)};
    }

    i64 GetBinaryOffset() const {
        return Profile_->stack_frames().binary_offset(*Index_);
    }

    TInlineChain GetInlineChain() const {
        i32 index = Profile_->stack_frames().inline_chain_id(*Index_);
        return TInlineChain{Profile_, TInlineChainId::FromInternalIndex(index)};
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "stack_frame");
        writer.Write("id", *GetIndex());

        writer.WriteKey("binary");
        GetBinary().DumpJson(writer);

        writer.Write("binary_offset", GetBinaryOffset());

        writer.WriteKey("inline_chain");
        GetInlineChain().DumpJson(writer);

        writer.CloseMap();
    }
};

class TStack : public TIndexedEntityReader<TStackId> {
public:
    using TBase::TBase;

    i32 GetStackFrameCount() const {
        auto [from, to] = GetOffsetRange(
            Profile_->stacks().offset(),
            Profile_->stacks().frame_id(),
            *Index_
        );

        return to - from;
    }

    TStackFrame GetStackFrame(i32 id) const {
        i32 position = id + Profile_->stacks().offset(*Index_);
        i32 index = Profile_->stacks().frame_id(position);
        return TStackFrame{Profile_, TStackFrameId::FromInternalIndex(index)};
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "stack");
        writer.Write("id", *GetIndex());
        writer.WriteKey("frames");
        writer.OpenArray();
        for (i32 i = 0; i < GetStackFrameCount(); ++i) {
            GetStackFrame(i).DumpJson(writer);
        }
        writer.CloseArray();
        writer.CloseMap();
    }
};

class TLabel : public TIndexedEntityReader<TLabelId> {
public:
    using TBase::TBase;

    bool IsString() const {
        return !IsNumber();
    }

    bool IsNumber() const {
        return GetTypeTag() == 1;
    }

    TStringRef GetKey() const {
        ui32 index = 0;
        if (IsNumber()) {
            index = Profile_->labels().numbers().key(GetPosition());
        } else {
            index = Profile_->labels().strings().key(GetPosition());
        }
        return {Profile_, index};
    }

    TStringRef GetString() const {
        Y_ENSURE(IsString());
        return GetStringUnsafe();
    }

    ui64 GetNumber() const {
        Y_ENSURE(IsNumber());
        return GetNumberUnsafe();
    }

    std::variant<TStringRef, i64> GetValue() const {
        if (IsString()) {
            return GetStringUnsafe();
        } else {
            return GetNumberUnsafe();
        }
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "label");
        writer.Write("id", *GetIndex());
        writer.Write("key", GetKey().View());
        if (IsNumber()) {
            writer.Write("value", GetNumberUnsafe());
        } else {
            writer.Write("value", GetStringUnsafe().View());
        }
        writer.CloseMap();
    }

private:
    i32 GetTypeTag() const {
        return *Index_ & 1;
    }

    i32 GetPosition() const {
        return *Index_ >> 1;
    }

    TStringRef GetStringUnsafe() const {
        Y_ASSERT(IsString());
        ui32 index = Profile_->labels().strings().value(GetPosition());
        return {Profile_, index};
    }

    i64 GetNumberUnsafe() const {
        Y_ASSERT(IsNumber());
        return Profile_->labels().numbers().value(GetPosition());
    }
};

class TThread : public TIndexedEntityReader<TThreadId> {
public:
    using TBase::TBase;

    ui32 GetThreadId() const {
        return Profile_->threads().thread_id(*Index_);
    }

    ui32 GetProcessId() const {
        return Profile_->threads().process_id(*Index_);
    }

    TStringRef GetThreadName() const {
        return {Profile_, Profile_->threads().thread_name(*Index_)};
    }

    TStringRef GetProcessName() const {
        return {Profile_, Profile_->threads().process_name(*Index_)};
    }

    i32 GetContainerCount() const {
        auto [begin, end] = GetOffsetRange(
            Profile_->threads().container_offset(),
            Profile_->threads().container_names(),
            *Index_
        );

        return end - begin;
    }

    TStringRef GetContainer(i32 id) const {
        ui32 offset = Profile_->threads().container_offset(*Index_);
        ui32 index = Profile_->threads().container_names(offset + id);
        return {Profile_, index};
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "thread");
        writer.Write("id", *GetIndex());
        writer.Write("thread_id", GetThreadId());
        writer.Write("process_id", GetProcessId());
        writer.Write("thread_name", GetThreadName().View());
        writer.Write("process_name", GetProcessName().View());
        writer.WriteKey("containers");
        writer.OpenArray();
        for (i32 i = 0; i < GetContainerCount(); ++i) {
            writer.Write(GetContainer(i).View());
        }
        writer.CloseArray();
        writer.CloseMap();
    }
};

class TSampleKey : public TIndexedEntityReader<TSampleKeyId> {
public:
    using TBase::TBase;

    TStack GetUserStack() const {
        return GetStack(Profile_->sample_keys().stacks().user_stack_id());
    }

    TStack GetKernelStack() const {
        return GetStack(Profile_->sample_keys().stacks().kernel_stack_id());
    }

    TThread GetThread() const {
        ui32 tid = Profile_->sample_keys().threads().thread_id(*Index_);
        return TThread{Profile_, TThreadId::FromInternalIndex(tid)};
    }

    i32 GetLabelCount() const {
        auto [from, to] = GetOffsetRange(
            Profile_->sample_keys().labels().first_label_id(),
            Profile_->sample_keys().labels().packed_label_id(),
            *Index_
        );

        return to - from;
    }

    TLabel GetLabel(i32 index) const {
        ui32 offset = Profile_->sample_keys().labels().first_label_id(*Index_);
        ui32 labelIndex = Profile_->sample_keys().labels().packed_label_id(offset + index);
        return TLabel{Profile_, labelIndex};
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "sample_key");
        writer.Write("id", *GetIndex());

        writer.WriteKey("kernel_stack");
        GetKernelStack().DumpJson(writer);

        writer.WriteKey("user_stack");
        GetUserStack().DumpJson(writer);

        writer.WriteKey("thread");
        GetThread().DumpJson(writer);

        writer.WriteKey("labels");
        writer.OpenArray();
        for (i32 i = 0; i < GetLabelCount(); ++i) {
            GetLabel(i).DumpJson(writer);
        }
        writer.CloseArray();

        writer.CloseMap();
    }

private:
    template <typename Stacks>
    TStack GetStack(Stacks&& stacks) const {
        i32 stackIndex = stacks.at(*Index_);
        return TStack{Profile_, TStackId::FromInternalIndex(stackIndex)};
    }
};

class TTimestamp {
public:
    TInstant AsInstantLossy() const {
        if (Seconds_ < 0) {
            return TInstant::Zero();
        }

        ui64 seconds = Seconds_;
        ui64 microseconds = seconds * 1'000'000 + Nanoseconds_ / 1000;
        if (microseconds / 1'000'000 != seconds) {
            return TInstant::Max();
        }

        return TInstant::MicroSeconds(microseconds);
    }

    i64 GetSeconds() const {
        return Seconds_;
    }

    double GetSecondsFloat() const {
        return static_cast<double>(Seconds_) + static_cast<double>(Nanoseconds_) * 1e-9;
    }

    ui32 GetNanosecondsOfSecond() const {
        return Nanoseconds_;
    }

private:
    i64 Seconds_ = 0;
    ui32 Nanoseconds_ = 0;
};

class TValueType : public TIndexedEntityReader<TValueTypeId> {
public:
    using TBase::TBase;

    TStringRef GetType() const {
        return {Profile_, GetTypeProto().type()};
    }

    TStringRef GetUnit() const {
        return {Profile_, GetTypeProto().unit()};
    }

private:
    const NProto::NProfile::ValueType& GetTypeProto() const {
        return Profile_->samples().values(*Index_).type();
    }
};

class TComment : public TIndexedEntityReader<TCommentId> {
public:
    using TBase::TBase;

    TStringRef GetString() const {
        return {Profile_, static_cast<ui32>(*Index_)};
    }
};

class TSample : public TIndexedEntityReader<TSampleId> {
public:
    using TBase::TBase;

    TSampleKey GetKey() const {
        ui32 keyIndex = Profile_->samples().key(*Index_);
        return TSampleKey{Profile_, TSampleKeyId::FromInternalIndex(keyIndex)};
    }

    i32 GetValueCount() const {
        return Profile_->samples().values_size();
    }

    ui64 GetValue(i32 index) const {
        Y_ASSERT(index < GetValueCount());
        return Profile_->samples().values(index).value(*Index_);
    }

    TValueType GetValueType(i32 index) const {
        Y_ASSERT(index < GetValueCount());
        return TValueType{Profile_, TValueTypeId::FromInternalIndex(index)};
    }

    TMaybe<google::protobuf::Timestamp> GetTimestamp() const {
        if (!Profile_->samples().has_timestamps()) {
            return Nothing();
        }

        Y_ABORT_UNLESS(false, "Unimplemented");
    }

    void DumpJson(NJson::TJsonWriter& writer) const {
        writer.OpenMap();
        writer.Write("kind", "sample");
        writer.Write("id", *GetIndex());

        writer.WriteKey("key");
        GetKey().DumpJson(writer);

        writer.WriteKey("timestamp");
        if (auto ts = GetTimestamp()) {
            writer.OpenMap();
            writer.Write("seconds", ts->seconds());
            writer.Write("nanoseconds", ts->nanos());
            writer.CloseMap();
        } else {
            writer.WriteNull();
        }

        writer.WriteKey("values");
        writer.OpenArray();
        for (i32 i = 0; i < GetValueCount(); ++i) {
            writer.OpenMap();
            writer.Write("value", GetValue(i));

            writer.WriteKey("type");
            writer.OpenMap();
            writer.Write("type", GetValueType(i).GetType().View());
            writer.Write("unit", GetValueType(i).GetUnit().View());
            writer.CloseMap();

            writer.CloseMap();
        }
        writer.CloseArray();

        writer.CloseMap();
    }
};

// Read-only representation of the profile.
class TProfile {
public:
    explicit TProfile(const NProto::NProfile::Profile* profile);

    ////////////////////////////////////////////////////////////////////////////////

    const NProto::NProfile::Metadata& GetMetadata() const;

    const NProto::NProfile::Features& GetFeatures() const;

    ////////////////////////////////////////////////////////////////////////////////

    TEntityArray<TStringRef> Strings() const {
        return {Profile_};
    }

    TEntityArray<TComment> Comments() const {
        return {Profile_};
    }

    TEntityArray<TValueType> ValueTypes() const {
        return {Profile_};
    }

    TEntityArray<TSample> Samples() const {
        return {Profile_};
    }

    TEntityArray<TSampleKey> SampleKeys() const {
        return {Profile_};
    }

    TEntityArray<TStack> Stacks() const {
        return {Profile_};
    }

    TEntityArray<TStackFrame> StackFrames() const {
        return {Profile_};
    }

    TEntityArray<TInlineChain> InlineChains() const {
        return {Profile_};
    }

    TEntityArray<TSourceLine> SourceLines() const {
        return {Profile_};
    }

    TEntityArray<TFunction> Functions() const {
        return {Profile_};
    }

    TEntityArray<TBinary> Binaries() const {
        return {Profile_};
    }

    TEntityArray<TThread> Threads() const {
        return {Profile_};
    }

    TEntityArray<TLabel> Labels() const {
        return {Profile_};
    }

    ////////////////////////////////////////////////////////////////////////////////

private:
    const NProto::NProfile::Profile* Profile_ = nullptr;
};

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
