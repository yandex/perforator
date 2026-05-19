#pragma once

#include "entity_index.h"

#include <perforator/proto/profile/profile.pb.h>
#include <perforator/proto/profile/well_known_labels.pb.h>

#include <util/datetime/base.h>
#include <util/generic/array_ref.h>
#include <util/generic/function_ref.h>
#include <util/generic/maybe.h>
#include <util/generic/yexception.h>

#include <optional>
#include <ranges>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

namespace NDetail {

// Range-returning methods MUST capture only what they use, by value (capture
// `[p = Profile_]`, never `this`/`*this`), so the returned range stays valid
// after a temporary reader is destroyed.

// Returns the [from, to) iota of indices into `values` for entry `rangeId` of
// an offsets + values pair stored as parallel arrays in the proto. `values` is
// used only for its `.size()`.
inline std::ranges::iota_view<i32, i32> OffsetRange(
    const std::ranges::sized_range auto& offsets,
    const std::ranges::sized_range auto& values,
    i32 rangeId
) {
    if (rangeId >= static_cast<i32>(offsets.size())) {
        return std::views::iota(0, 0);
    }
    i32 begin = offsets.at(rangeId);
    i32 end = (rangeId + 1 < static_cast<i32>(offsets.size()))
        ? offsets.at(rangeId + 1)
        : static_cast<i32>(values.size());
    return std::views::iota(begin, end);
}

// Dense [0, count) range of TEntity readers over the profile.
template <typename TEntity>
inline auto MakeEntityRange(const NProto::NProfile::Profile* p, i32 count) {
    return std::views::iota(0, count)
        | std::views::transform([p](i32 i) {
            return TEntity{p, TEntity::TIndex::FromInternalIndex(i)};
        });
}

} // namespace NDetail

// Well-known label key lookups. Pure over the WellKnownLabel enum (proto
// reflection, precomputed), no profile instance involved. Defined in
// profile.cpp; return views into static storage.

// Canonical key for a well-known label.
TStringBuf GetWellKnownLabelKey(NProto::NProfile::WellKnownLabel kind);

// Canonical + deprecated keys for a well-known label.
TConstArrayRef<TString> GetAllWellKnownLabelKeys(NProto::NProfile::WellKnownLabel kind);

// All well-known labels that have at least one key defined.
TConstArrayRef<NProto::NProfile::WellKnownLabel> GetWellKnownLabels();

////////////////////////////////////////////////////////////////////////////////

template <CStrongIndex Index>
class TIndexedEntityReader {
public:
    using TIndex = Index;
    using TBase = TIndexedEntityReader<Index>;

    TIndexedEntityReader(const NProto::NProfile::Profile* profile, ui32 id)
        : TIndexedEntityReader{profile, TIndex::FromInternalIndex(id)}
    {}

    TIndexedEntityReader(const NProto::NProfile::Profile* profile, Index id)
        : Profile_{profile}
        , Index_{id}
    {
        Y_ENSURE(id.IsValid());
    }

    Index GetIndex() const {
        return Index_;
    }

protected:
    const NProto::NProfile::Profile* Profile_ = nullptr;
    Index Index_;
};

////////////////////////////////////////////////////////////////////////////////

// Resolves a strtab string to a TStringBuf into proto-owned storage.
class TProfileString : public TIndexedEntityReader<TStringId> {
public:
    using TBase::TBase;

    TStringBuf View() const {
        const auto& strtab = Profile_->strtab();
        Y_ENSURE(strtab.offset_size() == strtab.length_size());
        Y_ENSURE(*Index_ < strtab.offset_size());
        ui32 offset = strtab.offset(*Index_);
        ui32 length = strtab.length(*Index_);
        TStringBuf strings = strtab.strings();
        Y_ENSURE(offset + length <= strings.size());
        return strings.SubString(offset, length);
    }

    explicit operator bool() const {
        return 0 != *Index_;
    }

    bool operator==(TStringBuf rhs) const {
        return View() == rhs;
    }

    bool operator!=(TStringBuf rhs) const {
        return !operator==(rhs);
    }
};

class TFunction : public TIndexedEntityReader<TFunctionId> {
public:
    using TBase::TBase;

    TProfileString GetName() const {
        return {Profile_, Profile_->functions().name(*Index_)};
    }

    TProfileString GetSystemName() const {
        return {Profile_, Profile_->functions().system_name(*Index_)};
    }

    TProfileString GetFileName() const {
        return {Profile_, Profile_->functions().filename(*Index_)};
    }

    ui32 GetStartLine() const {
        return Profile_->functions().start_line(*Index_);
    }

};

class TSourceLine : public TIndexedEntityReader<TSourceLineId> {
public:
    using TBase::TBase;

    TFunction GetFunction() const {
        i32 functionId = Profile_->inline_chains().lines().function_id(*Index_);
        return TFunction{Profile_, TFunctionId::FromInternalIndex(functionId)};
    }

    ui32 GetLine() const {
        return Profile_->inline_chains().lines().line(*Index_);
    }

    ui32 GetColumn() const {
        return Profile_->inline_chains().lines().column(*Index_);
    }

};

class TInlineChain : public TIndexedEntityReader<TInlineChainId> {
public:
    using TBase::TBase;

    auto GetLines() const {
        return NDetail::OffsetRange(
                Profile_->inline_chains().lines_offset(),
                Profile_->inline_chains().lines().function_id(),
                *Index_)
            | std::views::transform([p = Profile_](i32 pos) {
                return TSourceLine{p, TSourceLineId::FromInternalIndex(pos)};
            });
    }
};

class TBinary : public TIndexedEntityReader<TBinaryId> {
public:
    using TBase::TBase;

    TProfileString GetBuildId() const {
        ui32 id = Profile_->binaries().build_id(*Index_);
        return {Profile_, id};
    }

    TProfileString GetPath() const {
        ui32 id = Profile_->binaries().path(*Index_);
        return {Profile_, id};
    }

    bool HasSkewedAddresses() const {
        return Profile_->binaries().has_skewed_addresses(*Index_);
    }

};

class TStackFrame : public TIndexedEntityReader<TStackFrameId> {
public:
    using TBase::TBase;

    TBinary GetBinary() const {
        i32 index = Profile_->stack_frames().binary_id(*Index_);
        return TBinary{Profile_, TBinaryId::FromInternalIndex(index)};
    }

    ui64 GetAddress() const {
        return Profile_->stack_frames().address(*Index_);
    }

    TInlineChain GetInlineChain() const {
        i32 index = Profile_->stack_frames().inline_chain_id(*Index_);
        return TInlineChain{Profile_, TInlineChainId::FromInternalIndex(index)};
    }

};

class TStackSegment : public TIndexedEntityReader<TStackSegmentId> {
public:
    using TBase::TBase;

    auto GetFrames() const {
        return NDetail::OffsetRange(
                Profile_->stack_segments().frame_ids_offset(),
                Profile_->stack_segments().frame_ids(),
                *Index_)
            | std::views::transform([p = Profile_](i32 pos) {
                i32 frameId = p->stack_segments().frame_ids(pos);
                return TStackFrame{p, TStackFrameId::FromInternalIndex(frameId)};
            });
    }
};

class TStack : public TIndexedEntityReader<TStackId> {
public:
    using TBase::TBase;

    TStackFrame GetTopFrame() const {
        i32 id = Profile_->stacks().top_frame_id(*Index_);
        return TStackFrame{Profile_, TStackFrameId::FromInternalIndex(id)};
    }

    TStackSegment GetStackSegment() const {
        i32 id = Profile_->stacks().stack_segment_id(*Index_);
        return TStackSegment{Profile_, TStackSegmentId::FromInternalIndex(id)};
    }

    auto GetFrames() const {
        i32 topFrameId = Profile_->stacks().top_frame_id(*Index_);
        auto segRange = NDetail::OffsetRange(
            Profile_->stack_segments().frame_ids_offset(),
            Profile_->stack_segments().frame_ids(),
            Profile_->stacks().stack_segment_id(*Index_));
        return std::views::iota(0, 1 + static_cast<i32>(segRange.size()))
            | std::views::transform([p = Profile_, topFrameId, segRange](i32 i) {
                i32 frameId = (i == 0)
                    ? topFrameId
                    : p->stack_segments().frame_ids(segRange[i - 1]);
                return TStackFrame{p, TStackFrameId::FromInternalIndex(frameId)};
            });
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

    TProfileString GetKey() const {
        ui32 index = 0;
        if (IsNumber()) {
            index = Profile_->labels().numbers().key(GetPosition());
        } else {
            index = Profile_->labels().strings().key(GetPosition());
        }
        return {Profile_, index};
    }

    TProfileString GetString() const {
        Y_ENSURE(IsString());
        return GetStringUnsafe();
    }

    i64 GetNumber() const {
        Y_ENSURE(IsNumber());
        return GetNumberUnsafe();
    }

    std::variant<TProfileString, i64> GetValue() const {
        if (IsString()) {
            return GetStringUnsafe();
        } else {
            return GetNumberUnsafe();
        }
    }

private:
    i32 GetTypeTag() const {
        return *Index_ & 1;
    }

    i32 GetPosition() const {
        return *Index_ >> 1;
    }

    TProfileString GetStringUnsafe() const {
        Y_ASSERT(IsString());
        ui32 index = Profile_->labels().strings().value(GetPosition());
        return {Profile_, index};
    }

    i64 GetNumberUnsafe() const {
        Y_ASSERT(IsNumber());
        return Profile_->labels().numbers().value(GetPosition());
    }
};

class TLabelGroup : public TIndexedEntityReader<TLabelGroupId> {
public:
    using TBase::TBase;

    auto GetLabels() const {
        return NDetail::OffsetRange(
                Profile_->label_groups().packed_label_ids_offset(),
                Profile_->label_groups().packed_label_ids(),
                *Index_)
            | std::views::transform([p = Profile_](i32 pos) {
                return TLabel{p, p->label_groups().packed_label_ids(pos)};
            });
    }
};

class TSampleKey : public TIndexedEntityReader<TSampleKeyId> {
public:
    using TBase::TBase;

    auto GetStacks() const {
        return NDetail::OffsetRange(
                Profile_->sample_keys().stacks().stack_ids_offset(),
                Profile_->sample_keys().stacks().stack_ids(),
                *Index_)
            | std::views::transform([p = Profile_](i32 pos) {
                return TStack{p, p->sample_keys().stacks().stack_ids(pos)};
            });
    }

    TLabelGroup GetLabelGroup() const {
        ui32 id = Profile_->sample_keys().label_group_id(*Index_);
        return TLabelGroup{Profile_, TLabelGroupId::FromInternalIndex(id)};
    }

    // Labels attached directly to this sample key (not via the shared label group).
    auto GetDirectLabels() const {
        return NDetail::OffsetRange(
                Profile_->sample_keys().labels().packed_label_ids_offset(),
                Profile_->sample_keys().labels().packed_label_ids(),
                *Index_)
            | std::views::transform([p = Profile_](i32 pos) {
                return TLabel{p, p->sample_keys().labels().packed_label_ids(pos)};
            });
    }

    // Labels from the shared group followed by direct labels of this key.
    auto GetLabels() const {
        auto groupRange = NDetail::OffsetRange(
            Profile_->label_groups().packed_label_ids_offset(),
            Profile_->label_groups().packed_label_ids(),
            Profile_->sample_keys().label_group_id(*Index_));
        auto directRange = NDetail::OffsetRange(
            Profile_->sample_keys().labels().packed_label_ids_offset(),
            Profile_->sample_keys().labels().packed_label_ids(),
            *Index_);

        i32 total = static_cast<i32>(groupRange.size() + directRange.size());

        return std::views::iota(0, total)
            | std::views::transform([p = Profile_, groupRange, directRange](i32 i) {
                i32 groupCount = static_cast<i32>(groupRange.size());
                ui32 labelIndex = (i < groupCount)
                    ? p->label_groups().packed_label_ids(groupRange[i])
                    : p->sample_keys().labels().packed_label_ids(directRange[i - groupCount]);
                return TLabel{p, labelIndex};
            });
    }

    // Labels of this key matching a well-known kind (canonical name or any
    // deprecated alias). Group labels come first, since well-known labels
    // usually live in the shared group. Returns a filtered range so the
    // caller decides whether to take the first match or iterate all (e.g.
    // multi-valued "workload").
    auto GetWellKnownLabel(NProto::NProfile::WellKnownLabel kind) const {
        return GetLabels()
            | std::views::filter([keys = GetAllWellKnownLabelKeys(kind)](TLabel label) {
                TStringBuf key = label.GetKey().View();
                for (const TString& wellKnown : keys) {
                    if (key == wellKnown) {
                        return true;
                    }
                }
                return false;
            });
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

    TProfileString GetType() const {
        return {Profile_, GetTypeProto().type()};
    }

    TProfileString GetUnit() const {
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

    TProfileString GetString() const {
        return {Profile_, static_cast<ui32>(*Index_)};
    }
};

// Paired (value, value type) accessor for sample values. The proto is
// columnar: the per-sample value array is co-indexed with the profile's
// value types, so ValueIndex_ is also the value type's id.
class TSampleValue {
public:
    TSampleValue(const NProto::NProfile::Profile* profile, TSampleId sampleId, i32 valueIndex)
        : Profile_{profile}
        , SampleId_{sampleId}
        , ValueIndex_{valueIndex}
    {}

    TValueType GetValueType() const {
        return TValueType{Profile_, TValueTypeId::FromInternalIndex(ValueIndex_)};
    }

    ui64 GetValue() const {
        return Profile_->samples().values(ValueIndex_).value(*SampleId_);
    }

private:
    const NProto::NProfile::Profile* Profile_;
    TSampleId SampleId_;
    i32 ValueIndex_;
};

class TSample : public TIndexedEntityReader<TSampleId> {
public:
    using TBase::TBase;

    TSampleKey GetKey() const {
        ui32 keyIndex = Profile_->samples().key(*Index_);
        return TSampleKey{Profile_, TSampleKeyId::FromInternalIndex(keyIndex)};
    }

    auto GetValues() const {
        return std::views::iota(0, Profile_->samples().values_size())
            | std::views::transform([p = Profile_, sampleId = Index_](i32 i) {
                return TSampleValue{p, sampleId, i};
            });
    }

    TMaybe<google::protobuf::Timestamp> GetProtoTimestamp() const {
        if (!Profile_->samples().has_timestamps()) {
            return Nothing();
        }

        auto start = Profile_->samples().timestamps().start_timestamp();
        auto delta = Profile_->samples().timestamps().delta_nanoseconds(*Index_);

        static constexpr i64 nanosecondsInSecond = 1'000'000'000;
        i64 deltaSeconds = delta / nanosecondsInSecond;
        i64 deltaNanoseconds = delta % nanosecondsInSecond;
        if (deltaNanoseconds < 0) {
            deltaNanoseconds += nanosecondsInSecond;
            Y_ASSERT(deltaNanoseconds >= 0);
            deltaSeconds -= 1;
        }

        i64 seconds = start.seconds() + deltaSeconds;
        i64 nanos = i64{start.nanos()} + deltaNanoseconds;
        if (nanos >= deltaNanoseconds) {
            nanos -= deltaNanoseconds;
            seconds += 1;
        }
        Y_ASSERT(nanos >= 0);
        Y_ASSERT(nanos < nanosecondsInSecond);

        google::protobuf::Timestamp ts;
        ts.set_seconds(seconds);
        ts.set_nanos(nanos);
        return ts;
    }

    TMaybe<TInstant> GetInstantTimestamp() const {
        auto ts = GetProtoTimestamp();
        if (!ts) {
            return Nothing();
        }

        ui64 micros = ts->seconds() * 1'000'000 + ts->nanos() / 1000;
        return TInstant::MicroSeconds(micros);
    }

};

// Read-only representation of the profile.
class TProfile {
public:
    explicit TProfile(const NProto::NProfile::Profile* profile);

    ////////////////////////////////////////////////////////////////////////////////

    const NProto::NProfile::Metadata& GetMetadata() const;

    ////////////////////////////////////////////////////////////////////////////////
    // Single-entity accessors.

    TProfileString String(TStringId id) const { return {Profile_, id}; }
    TComment Comment(TCommentId id) const { return {Profile_, id}; }
    TValueType ValueType(TValueTypeId id) const { return {Profile_, id}; }
    TSample Sample(TSampleId id) const { return {Profile_, id}; }
    TSampleKey SampleKey(TSampleKeyId id) const { return {Profile_, id}; }
    TStack Stack(TStackId id) const { return {Profile_, id}; }
    TStackFrame StackFrame(TStackFrameId id) const { return {Profile_, id}; }
    TStackSegment StackSegment(TStackSegmentId id) const { return {Profile_, id}; }
    TInlineChain InlineChain(TInlineChainId id) const { return {Profile_, id}; }
    TSourceLine SourceLine(TSourceLineId id) const { return {Profile_, id}; }
    TFunction Function(TFunctionId id) const { return {Profile_, id}; }
    TBinary Binary(TBinaryId id) const { return {Profile_, id}; }
    TLabelGroup LabelGroup(TLabelGroupId id) const { return {Profile_, id}; }
    TLabel Label(TLabelId id) const { return {Profile_, id}; }

    ////////////////////////////////////////////////////////////////////////////////
    // Entity ranges.

    auto Strings() const {
        return NDetail::MakeEntityRange<TProfileString>(Profile_, Profile_->strtab().length_size());
    }

    auto Comments() const {
        return NDetail::MakeEntityRange<TComment>(Profile_, Profile_->comments().comment_size());
    }

    auto ValueTypes() const {
        return NDetail::MakeEntityRange<TValueType>(Profile_, Profile_->samples().values_size());
    }

    auto Samples() const {
        return NDetail::MakeEntityRange<TSample>(Profile_, Profile_->samples().key_size());
    }

    auto SampleKeys() const {
        return NDetail::MakeEntityRange<TSampleKey>(Profile_, Profile_->sample_keys().stacks().stack_ids_offset_size());
    }

    auto Stacks() const {
        return NDetail::MakeEntityRange<TStack>(Profile_, Profile_->stacks().top_frame_id_size());
    }

    auto StackFrames() const {
        return NDetail::MakeEntityRange<TStackFrame>(Profile_, Profile_->stack_frames().binary_id_size());
    }

    auto StackSegments() const {
        return NDetail::MakeEntityRange<TStackSegment>(Profile_, Profile_->stack_segments().frame_ids_offset_size());
    }

    auto InlineChains() const {
        return NDetail::MakeEntityRange<TInlineChain>(Profile_, Profile_->inline_chains().lines_offset_size());
    }

    auto SourceLines() const {
        return NDetail::MakeEntityRange<TSourceLine>(Profile_, Profile_->inline_chains().lines().function_id_size());
    }

    auto Functions() const {
        return NDetail::MakeEntityRange<TFunction>(Profile_, Profile_->functions().name_size());
    }

    auto Binaries() const {
        return NDetail::MakeEntityRange<TBinary>(Profile_, Profile_->binaries().build_id_size());
    }

    auto LabelGroups() const {
        return NDetail::MakeEntityRange<TLabelGroup>(Profile_, Profile_->label_groups().packed_label_ids_offset_size());
    }

    // Labels are iterated in ascending packed-index order. Packed index has
    // bit 0 = type tag (0 = string, 1 = number) and bits 1+ = position, so the
    // valid set interleaves up to `2*Min(sc, nc+1) - 1`, then continues on the
    // longer side with stride 2.
    auto Labels() const {
        i32 sc = Profile_->labels().strings().key_size();
        i32 nc = Profile_->labels().numbers().key_size();
        i32 cap = Min(sc * 2, nc * 2 + 1) - 1;
        return std::views::iota(0, sc + nc)
            | std::views::transform([p = Profile_, cap](i32 index) {
                i32 packed = index < cap ? index : 2 * index - cap;
                return TLabel{p, TLabelId::FromInternalIndex(packed)};
            });
    }

    ////////////////////////////////////////////////////////////////////////////////

private:
    const NProto::NProfile::Profile* Profile_ = nullptr;
};

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
