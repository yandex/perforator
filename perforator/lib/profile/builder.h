#pragma once

#include "entity_index.h"

#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/containers/absl/flat_hash_set.h>
#include <library/cpp/containers/stack_vector/stack_vec.h>
#include <library/cpp/introspection/introspection.h>

#include <util/datetime/base.h>
#include <util/digest/city.h>
#include <util/digest/multi.h>
#include <util/generic/strbuf.h>

#include <optional>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

#define Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(Self) \
    bool operator==(const Self&) const noexcept = default; \
    bool operator!=(const Self&) const noexcept = default;

#define Y_DEFAULT_ABSL_HASHABLE_TYPE(Self) \
    template <typename H> \
    friend H AbslHashValue(H hash, const Self& self) { \
        return H::combine(std::move(hash), NIntrospection::Members(self)); \
    }

template <typename A>
ui64 HashArrayFast(A&& array) {
    static_assert(std::has_unique_object_representations_v<std::decay_t<decltype(array[0])>>);
    return CityHash64(reinterpret_cast<const char*>(array.data()), array.size() * sizeof(array[0]));
}

struct TValueTypeInfo {
    TStringId Type;
    TStringId Unit;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TValueTypeInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TValueTypeInfo);
};

struct TStringLabelInfo {
    TStringId Key;
    TStringId Value;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TStringLabelInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TStringLabelInfo);
};

struct TNumberLabelInfo {
    TStringId Key;
    i64 Value;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TNumberLabelInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TNumberLabelInfo);
};

struct TBinaryInfo {
    TStringId BuildId = TStringId::Zero();
    TStringId Path = TStringId::Zero();
    bool HasSkewedAddresses = false;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TBinaryInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TBinaryInfo);
};

struct TFunctionInfo {
    TStringId Name = TStringId::Zero();
    TStringId SystemName = TStringId::Zero();
    TStringId FileName = TStringId::Zero();
    ui32 StartLine = 0;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TFunctionInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TFunctionInfo);
};

struct TSourceLineInfo {
    TFunctionId Function = TFunctionId::Zero();
    ui32 Line = 0;
    ui32 Column = 0;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TSourceLineInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TSourceLineInfo);
};

struct TInlineChainInfo {
    TSmallVec<TSourceLineInfo> Lines;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TInlineChainInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TInlineChainInfo);

    ui64 StableHashValue() const {
        return HashArrayFast(Lines);
    }
};

struct TStackFrameInfo {
    TBinaryId Binary = TBinaryId::Zero();
    ui64 Address = 0;
    TInlineChainId InlineChain = TInlineChainId::Zero();

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TStackFrameInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TStackFrameInfo);
};

struct TStackSegmentInfo {
    TStackVec<TStackFrameId, 64> Stack;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TStackSegmentInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TStackSegmentInfo);

    ui64 StableHashValue() const {
        return HashArrayFast(Stack);
    }
};

struct TStackInfo {
    TStackFrameId TopFrame = TStackFrameId::Zero();
    TStackSegmentId StackSegment = TStackSegmentId::Zero();

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TStackInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TStackInfo);
};

struct TLabelGroupInfo {
    TStackVec<TLabelId, 8> Labels;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TLabelGroupInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TLabelGroupInfo);

    ui64 StableHashValue() const {
        return HashArrayFast(Labels);
    }
};

struct TSampleKeyInfo {
    TLabelGroupId LabelGroup = TLabelGroupId::Zero();
    TStackVec<TStackId, 8> Stacks;
    TStackVec<TLabelId, 8> Labels;

    ui64 StableHashValue() const {
        return MultiHash(
            *LabelGroup,
            HashArrayFast(Stacks),
            HashArrayFast(Labels)
        );
    }

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TSampleKeyInfo);

    template <typename H>
    friend H AbslHashValue(H state, const TSampleKeyInfo& self) {
        state = H::combine(std::move(state), self.LabelGroup);

        state = H::combine_contiguous(std::move(state),
            self.Stacks.data(),
            self.Stacks.size()
        );

        return H::combine_contiguous(std::move(state),
            self.Labels.data(),
            self.Labels.size()
        );
    }
};

struct TSampleTimestamp {
    i64 Seconds = 0;
    ui32 NanoSeconds = 0;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TSampleTimestamp);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TSampleTimestamp);
};

struct TSampleInfo {
    TSampleKeyId Key = TSampleKeyId::Zero();
    std::optional<TSampleTimestamp> Timestamp;
    TStackVec<std::pair<TValueTypeId, ui64>, 4> Values;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TSampleInfo);
    Y_DEFAULT_ABSL_HASHABLE_TYPE(TSampleInfo);
};

////////////////////////////////////////////////////////////////////////////////

// TProfileBuilder is a write-only low-level builder of a profile.
class TProfileBuilder {
public:
    // A bunch of forward declarations to make the class readable.
    class TMetadataBuilder;
    class TBinaryBuilder;
    class TFunctionBuilder;
    class TInlineChainBuilder;
    class TStackFrameBuilder;
    class TStackSegmentBuilder;
    class TLabelGroupBuilder;
    class TStackBuilder;
    class TSampleKeyBuilder;
    class TSimpleSampleKeyBuilder;
    class TSampleBuilder;

public:
    explicit TProfileBuilder(NProto::NProfile::Profile* profile);
    ~TProfileBuilder();

    TMetadataBuilder Metadata();

    TStringId AddString(TStringBuf string);

    TCommentId AddComment(TStringBuf string);
    TCommentId AddComment(TStringId string);

    TValueTypeId AddValueType(TStringBuf type, TStringBuf unit);
    TValueTypeId AddValueType(TStringId type, TStringId unit);

    TLabelId AddStringLabel(TStringBuf key, TStringBuf value);
    TLabelId AddStringLabel(TStringId key, TStringId value);
    TLabelId AddNumericLabel(TStringBuf key, i64 value);
    TLabelId AddNumericLabel(TStringId key, i64 value);

    TBinaryBuilder AddBinary();
    TBinaryId AddBinary(const TBinaryInfo& key, const TBinaryInfo& value);

    TFunctionBuilder AddFunction();
    TFunctionId AddFunction(const TFunctionInfo& info);

    TInlineChainBuilder AddInlineChain();
    TInlineChainId AddInlineChain(const TInlineChainInfo& info);

    TStackFrameBuilder AddStackFrame();
    TStackFrameId AddStackFrame(const TStackFrameInfo& info);

    TStackSegmentBuilder AddStackSegment();
    TStackSegmentId AddStackSegment(const TStackSegmentInfo& info);

    TStackBuilder AddStack();
    TStackId AddStack(const TStackInfo& info);

    TLabelGroupBuilder AddLabelGroup();
    TLabelGroupId AddLabelGroup(const TLabelGroupInfo& info);

    TSampleKeyBuilder AddSampleKey();
    TSampleKeyId AddSampleKey(const TSampleKeyInfo& info);

    TSimpleSampleKeyBuilder AddSimpleSampleKey();

    TSampleBuilder AddSample();
    TSampleId AddSample(const TSampleInfo& info);

    NProto::NProfile::Profile* Finish() &&;

public:
    class TMetadataBuilder {
    public:
        TMetadataBuilder(TProfileBuilder& builder, NProto::NProfile::Metadata& meta)
            : Builder_{builder}
            , Metadata_{meta}
        {}

        TMetadataBuilder& SetHostname(TStringBuf hostname) {
            Metadata_.set_hostname(Builder_.AddString(hostname).GetInternalIndex());
            return *this;
        }

        NProto::NProfile::Metadata& GetProto() const {
            return Metadata_;
        }

        TProfileBuilder& Finish() {
            return Builder_;
        }

    private:
        TProfileBuilder& Builder_;
        NProto::NProfile::Metadata& Metadata_;
    };

    class TBinaryBuilder {
    public:
        TBinaryBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TBinaryBuilder& SetBuildId(TStringBuf id) {
            return SetBuildId(Builder_.AddString(id));
        }

        TBinaryBuilder& SetBuildId(TStringId id) {
            Info_.BuildId = id;
            return *this;
        }

        TBinaryBuilder& SetPath(TStringBuf path) {
            return SetPath(Builder_.AddString(path));
        }

        TBinaryBuilder& SetPath(TStringId path) {
            Info_.Path = path;
            return *this;
        }

        TBinaryBuilder& SetIgnoreBinaryPaths(bool value) {
            IgnoreBinaryPaths_ = value;
            return *this;
        }

        TBinaryBuilder& SetHasSkewedAddresses(bool value) {
            Info_.HasSkewedAddresses = value;
            return *this;
        }

        TBinaryId Finish() {
            if (IgnoreBinaryPaths_) {
                return Builder_.AddBinary(TBinaryInfo {
                    .BuildId = Info_.BuildId,
                    .Path = Info_.BuildId == TStringId::Zero() ? Info_.Path : TStringId::Zero(),
                }, Info_);
            } else {
                return Builder_.AddBinary(Info_, Info_);
            }
        }

    private:
        bool IgnoreBinaryPaths_ = false;
        TBinaryInfo Info_;
        TProfileBuilder& Builder_;
    };

    class TFunctionBuilder {
    public:
        TFunctionBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TFunctionBuilder& SetName(TStringBuf name) {
            return SetName(Builder_.AddString(name));
        }

        TFunctionBuilder& SetName(TStringId name) {
            Info_.Name = name;
            return *this;
        }

        TFunctionBuilder& SetSystemName(TStringBuf name) {
            return SetSystemName(Builder_.AddString(name));
        }

        TFunctionBuilder& SetSystemName(TStringId name) {
            Info_.SystemName = name;
            return *this;
        }

        TFunctionBuilder& SetFileName(TStringBuf name) {
            return SetFileName(Builder_.AddString(name));
        }

        TFunctionBuilder& SetFileName(TStringId name) {
            Info_.FileName = name;
            return *this;
        }

        TFunctionBuilder& SetStartLine(ui32 line) {
            Info_.StartLine = line;
            return *this;
        }

        TFunctionId Finish() {
            return Builder_.AddFunction(Info_);
        }

    private:
        TFunctionInfo Info_;
        TProfileBuilder& Builder_;
    };

    class TSourceLineBuilder {
    public:
        TSourceLineBuilder(
            TInlineChainBuilder& builder,
            TSourceLineInfo& info
        )
            : Builder_{builder}
            , Info_{info}
        {}

        TSourceLineBuilder& SetFunction(TFunctionId function) {
            Info_.Function = function;
            return *this;
        }

        TSourceLineBuilder& SetLine(ui32 line) {
            Info_.Line = line;
            return *this;
        }

        TSourceLineBuilder& SetColumn(ui32 column) {
            Info_.Column = column;
            return *this;
        }

        TInlineChainBuilder& Finish() {
            return Builder_;
        }

    private:
        TInlineChainBuilder& Builder_;
        TSourceLineInfo& Info_;
    };

    class TInlineChainBuilder {
    public:
        TInlineChainBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TSourceLineBuilder AddLine() {
            return TSourceLineBuilder{*this, Info_.Lines.emplace_back()};
        }

        TInlineChainId Finish() {
            return Builder_.AddInlineChain(Info_);
        }

    private:
        TInlineChainInfo Info_;
        TProfileBuilder& Builder_;
    };

    class TStackFrameBuilder {
    public:
        TStackFrameBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TStackFrameBuilder& SetBinary(TBinaryId binary) {
            Info_.Binary = binary;
            return *this;
        }

        TStackFrameBuilder& SetAddress(ui64 address) {
            Info_.Address = address;
            return *this;
        }

        TStackFrameBuilder& SetInlineChain(TInlineChainId sloc) {
            Info_.InlineChain = sloc;
            return *this;
        }

        TStackFrameId Finish() {
            return Builder_.AddStackFrame(Info_);
        }

    private:
        TStackFrameInfo Info_;
        TProfileBuilder& Builder_;
    };

    class TStackSegmentBuilder {
    public:
        TStackSegmentBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TStackSegmentBuilder& AddFrame(TStackFrameId frame) {
            Info_.Stack.push_back(frame);
            return *this;
        }

        TStackSegmentId Finish() {
            return Builder_.AddStackSegment(Info_);
        }

    private:
        TStackSegmentInfo Info_;
        TProfileBuilder& Builder_;
    };

    class TStackBuilder {
    public:
        TStackBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TStackBuilder& SetTopFrame(TStackFrameId frame) {
            Info_.TopFrame = frame;
            return *this;
        }

        TStackBuilder& SetStackSegment(TStackSegmentId segment) {
            Info_.StackSegment = segment;
            return *this;
        }

        TStackId Finish() {
            return Builder_.AddStack(Info_);
        }

    private:
        TStackInfo Info_;
        TProfileBuilder& Builder_;
    };

    class TLabelGroupBuilder {
    public:
        TLabelGroupBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TLabelGroupBuilder& AddLabel(TLabelId label) {
            Info_.Labels.push_back(label);
            return *this;
        }

        TLabelGroupId Finish() {
            return Builder_.AddLabelGroup(Info_);
        }

    private:
        TLabelGroupInfo Info_;
        TProfileBuilder& Builder_;
    };

    class TSampleKeyBuilder {
    public:
        TSampleKeyBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TSampleKeyBuilder& AddLabel(TLabelId label) {
            Info_.Labels.push_back(label);
            return *this;
        }

        TSampleKeyBuilder& SetLabelGroup(TLabelGroupId labelGroup) {
            Info_.LabelGroup = labelGroup;
            return *this;
        }

        TSampleKeyBuilder& AddStack(TStackId stack) {
            Info_.Stacks.push_back(stack);
            return *this;
        }

        TSampleKeyId Finish() {
            return Builder_.AddSampleKey(Info_);
        }

    private:
        TSampleKeyInfo Info_;
        TProfileBuilder& Builder_;
    };

    class TSimpleSampleKeyBuilder {
    public:
        TSimpleSampleKeyBuilder(TProfileBuilder& builder)
            : KeyBuilder_{builder}
            , Builder_{builder}
        {}

        TSimpleSampleKeyBuilder& AddLabel(TLabelId label);

        TSimpleSampleKeyBuilder& AddFrame(TStackFrameId frame);

        TSampleKeyId Finish() {
            if (TopFrame_ != TStackFrameId::Invalid()) {
                auto builder = Builder_.AddStack();
                builder.SetTopFrame(TopFrame_);
                if (!StackSegment_.Stack.empty()) {
                    builder.SetStackSegment(Builder_.AddStackSegment(StackSegment_));
                }
                KeyBuilder_.AddStack(builder.Finish());
            }
            if (LabelGroup_.Labels) {
                KeyBuilder_.SetLabelGroup(Builder_.AddLabelGroup(LabelGroup_));
            }
            return KeyBuilder_.Finish();
        }

    private:
        TSampleKeyBuilder KeyBuilder_;
        TLabelGroupInfo LabelGroup_;
        TStackFrameId TopFrame_ = TStackFrameId::Invalid();
        TStackSegmentInfo StackSegment_;
        TProfileBuilder& Builder_;
    };

    class TSampleBuilder {
    public:
        TSampleBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TSampleBuilder& SetSampleKey(TSampleKeyId key) {
            Info_.Key = key;
            return *this;
        }

        TSampleBuilder& AddValue(TValueTypeId idx, ui64 value) {
            Info_.Values.push_back({idx, value});
            return *this;
        }

        TSampleBuilder& SetTimestamp(TInstant ts) {
            return SetTimestamp(ts.Seconds(), ts.NanoSecondsOfSecond());
        }

        TSampleBuilder& SetTimestamp(i64 seconds, ui32 nanoseconds) {
            Info_.Timestamp = TSampleTimestamp{
                .Seconds = seconds,
                .NanoSeconds = nanoseconds,
            };
            return *this;
        }

        TProfileBuilder& Finish() {
            Builder_.AddSample(Info_);
            return Builder_;
        }

    private:
        TSampleInfo Info_;
        TProfileBuilder& Builder_;
    };


private:
    class TImpl;
    THolder<TImpl> Impl_;
};

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
