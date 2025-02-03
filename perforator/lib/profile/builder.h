#pragma once

#include "entity_index.h"

#include <perforator/proto/profile/profile.pb.h>

#include <library/cpp/containers/stack_vector/stack_vec.h>
#include <library/cpp/introspection/introspection.h>

#include <util/datetime/base.h>
#include <util/digest/multi.h>
#include <util/generic/strbuf.h>

#include <optional>


namespace NPerforator::NProfile {

////////////////////////////////////////////////////////////////////////////////

#define Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(Self) \
    bool operator==(const Self&) const noexcept = default; \
    bool operator!=(const Self&) const noexcept = default;

#define Y_DEFAULT_HASHABLE_TYPE(Self) \
    template <typename H> \
    friend H AbslHashValue(H hash, const Self& self) { \
        return H::combine(std::move(hash), NIntrospection::Members(self)); \
    }

struct TValueTypeInfo {
    TStringId Type;
    TStringId Unit;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TValueTypeInfo);
    Y_DEFAULT_HASHABLE_TYPE(TValueTypeInfo);
};

struct TStringLabelInfo {
    TStringId Key;
    TStringId Value;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TStringLabelInfo);
    Y_DEFAULT_HASHABLE_TYPE(TStringLabelInfo);
};

struct TNumberLabelInfo {
    TStringId Key;
    i64 Value;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TNumberLabelInfo);
    Y_DEFAULT_HASHABLE_TYPE(TNumberLabelInfo);
};

struct TThreadInfo {
    ui64 ProcessId = 0;
    ui64 ThreadId = 0;
    TStringId ProcessNameIdx = TStringId::Zero();
    TStringId ThreadNameIdx = TStringId::Zero();
    TStackVec<TStringId, 8> ContainerIdx;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TThreadInfo);
    Y_DEFAULT_HASHABLE_TYPE(TThreadInfo);
};

struct TBinaryInfo {
    TStringId BuildId = TStringId::Zero();
    TStringId Path = TStringId::Zero();

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TBinaryInfo);
    Y_DEFAULT_HASHABLE_TYPE(TBinaryInfo);
};

struct TFunctionInfo {
    TStringId Name = TStringId::Zero();
    TStringId SystemName = TStringId::Zero();
    TStringId FileName = TStringId::Zero();
    ui32 StartLine = 0;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TFunctionInfo);
    Y_DEFAULT_HASHABLE_TYPE(TFunctionInfo);
};

struct TSourceLineInfo {
    TFunctionId Function = TFunctionId::Zero();
    ui32 Line = 0;
    ui32 Column = 0;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TSourceLineInfo);
    Y_DEFAULT_HASHABLE_TYPE(TSourceLineInfo);
};

struct TInlineChainInfo {
    TSmallVec<TSourceLineInfo> Lines;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TInlineChainInfo);
    Y_DEFAULT_HASHABLE_TYPE(TInlineChainInfo);
};

struct TStackFrameInfo {
    TBinaryId Binary = TBinaryId::Zero();
    i64 BinaryOffset = 0;
    TInlineChainId InlineChain = TInlineChainId::Zero();

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TStackFrameInfo);
    Y_DEFAULT_HASHABLE_TYPE(TStackFrameInfo);
};

struct TStackInfo {
    TStackVec<TStackFrameId, 128> Stack;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TStackInfo);
    Y_DEFAULT_HASHABLE_TYPE(TStackInfo);
};

struct TSampleKeyInfo {
    TThreadId Thread = TThreadId::Zero();
    TStackId UserStack = TStackId::Zero();
    TStackId KernelStack = TStackId::Zero();
    TStackVec<TLabelId, 8> Labels;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TSampleKeyInfo);

    template <typename H>
    friend H AbslHashValue(H state, const TSampleKeyInfo& self) {
        state = H::combine(std::move(state),
            self.Thread,
            self.UserStack,
            self.KernelStack
        );
        return H::combine_unordered(std::move(state),
            self.Labels.begin(),
            self.Labels.end()
        );
    }
};

struct TSampleTimestamp {
    i64 Seconds = 0;
    ui32 NanoSeconds = 0;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TSampleTimestamp);
    Y_DEFAULT_HASHABLE_TYPE(TSampleTimestamp);
};

struct TSampleInfo {
    TSampleKeyId Key = TSampleKeyId::Zero();
    std::optional<TSampleTimestamp> Timestamp;
    TStackVec<std::pair<TValueTypeId, ui64>, 4> Values;

    Y_DEFAULT_EQUALITY_COMPARABLE_TYPE(TSampleInfo);
    Y_DEFAULT_HASHABLE_TYPE(TSampleInfo);
};

////////////////////////////////////////////////////////////////////////////////

// TProfileBuilder is a write-only low-level builder of a profile.
class TProfileBuilder {
public:
    // A bunch of forward declarations to make the class readable.
    class TMetadataBuilder;
    class TFeaturesBuilder;
    class TThreadBuilder;
    class TBinaryBuilder;
    class TFunctionBuilder;
    class TInlineChainBuilder;
    class TStackFrameBuilder;
    class TStackBuilder;
    class TSampleKeyBuilder;
    class TSampleBuilder;

public:
    explicit TProfileBuilder(NProto::NProfile::Profile* profile);
    ~TProfileBuilder();

    TMetadataBuilder Metadata();
    TFeaturesBuilder Features();

    TStringId AddString(TStringBuf string);

    TCommentId AddComment(TStringBuf string);
    TCommentId AddComment(TStringId string);

    TValueTypeId AddValueType(TStringBuf type, TStringBuf unit);
    TValueTypeId AddValueType(TStringId type, TStringId unit);

    TLabelId AddStringLabel(TStringBuf key, TStringBuf value);
    TLabelId AddStringLabel(TStringId key, TStringId value);
    TLabelId AddNumericLabel(TStringBuf key, i64 value);
    TLabelId AddNumericLabel(TStringId key, i64 value);

    TThreadBuilder AddThread();
    TThreadId AddThread(const TThreadInfo& info);

    TBinaryBuilder AddBinary();
    TBinaryId AddBinary(const TBinaryInfo& info);

    TFunctionBuilder AddFunction();
    TFunctionId AddFunction(const TFunctionInfo& info);

    TInlineChainBuilder AddInlineChain();
    TInlineChainId AddInlineChain(const TInlineChainInfo& info);

    TStackFrameBuilder AddStackFrame();
    TStackFrameId AddStackFrame(const TStackFrameInfo& info);

    TStackBuilder AddStack();
    TStackId AddStack(const TStackInfo& info);

    TSampleKeyBuilder AddSampleKey();
    TSampleKeyId AddSampleKey(const TSampleKeyInfo& info);

    TSampleBuilder AddSample();
    TSampleId AddSample(const TSampleInfo& info);

    void Finish();

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

    class TFeaturesBuilder {
    public:
        TFeaturesBuilder(TProfileBuilder& builder, NProto::NProfile::Features& features)
            : Builder_{builder}
            , Features_{features}
        {}

        TFeaturesBuilder& SetHasSkewedBinaryOffsets(bool has) {
            Features_.set_has_skewed_binary_offsets(has);
            return *this;
        }

        NProto::NProfile::Features& GetProto() const {
            return Features_;
        }

        TProfileBuilder& Finish() {
            return Builder_;
        }

    private:
        TProfileBuilder& Builder_;
        NProto::NProfile::Features& Features_;
    };

    class TThreadBuilder {
    public:
        TThreadBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TThreadBuilder& SetProcessId(ui32 id) {
            Info_.ProcessId = id;
            return *this;
        }

        TThreadBuilder& SetThreadId(ui32 id) {
            Info_.ThreadId = id;
            return *this;
        }

        TThreadBuilder& SetProcessName(TStringBuf name) {
            return SetProcessName(Builder_.AddString(name));
        }

        TThreadBuilder& SetProcessName(TStringId name) {
            Info_.ProcessNameIdx = name;
            return *this;
        }

        TThreadBuilder& SetThreadName(TStringBuf name) {
            return SetThreadName(Builder_.AddString(name));
        }

        TThreadBuilder& SetThreadName(TStringId name) {
            Info_.ThreadNameIdx = name;
            return *this;
        }

        TThreadBuilder& AddContainerName(TStringBuf name) {
            return AddContainerName(Builder_.AddString(name));
        }

        TThreadBuilder& AddContainerName(TStringId name) {
            Info_.ContainerIdx.push_back(name);
            return *this;
        }

        TThreadId Finish() {
            return Builder_.AddThread(Info_);
        }

    private:
        TThreadInfo Info_;
        TProfileBuilder& Builder_;
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

        TBinaryId Finish() {
            return Builder_.AddBinary(Info_);
        }

    private:
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

        TStackFrameBuilder& SetBinaryOffset(i64 offset) {
            Info_.BinaryOffset = offset;
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

    class TStackBuilder {
    public:
        TStackBuilder(TProfileBuilder& builder)
            : Builder_{builder}
        {}

        TStackBuilder& AddStackFrame(TStackFrameId frame) {
            Info_.Stack.push_back(frame);
            return *this;
        }

        TStackId Finish() {
            return Builder_.AddStack(Info_);
        }

    private:
        TStackInfo Info_;
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

        TSampleKeyBuilder& SetThread(TThreadId thread) {
            Info_.Thread = thread;
            return *this;
        }

        TSampleKeyBuilder& SetUserStack(TStackId stack) {
            Info_.UserStack = stack;
            return *this;
        }

        TSampleKeyBuilder& SetKernelStack(TStackId stack) {
            Info_.KernelStack = stack;
            return *this;
        }

        TSampleKeyId Finish() {
            return Builder_.AddSampleKey(Info_);
        }

    private:
        TSampleKeyInfo Info_;
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

} // namespace NProfile::NProfile
