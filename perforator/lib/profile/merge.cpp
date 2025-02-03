#include "builder.h"
#include "entity_index.h"
#include "merge.h"
#include "profile.h"

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>

#include <util/system/mutex.h>

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

class TSingleProfileMerger {
public:
    TSingleProfileMerger(
        TProfileBuilder& builder,
        TMergeOptions options,
        TProfile profile,
        ui32 profileIndex
    )
        : Builder_{builder}
        , Options_{options}
        , Profile_{profile}
        , IsFirstProfile_{profileIndex == 0}
        , Strings_{*Profile_.Strings().GetPastTheEndIndex()}
        , ValueTypes_{*Profile_.ValueTypes().GetPastTheEndIndex()}
        , Samples_{*Profile_.Samples().GetPastTheEndIndex()}
        , SampleKeys_{*Profile_.SampleKeys().GetPastTheEndIndex()}
        , Stacks_{*Profile_.Stacks().GetPastTheEndIndex()}
        , Binaries_{*Profile_.Binaries().GetPastTheEndIndex()}
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
            MergeSample(sample);
        }
    }

    void MergeSample(TSample sample) {
        auto builder = Builder_.AddSample();

        if (auto ts = sample.GetTimestamp(); ts && Options_.KeepTimestamps) {
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
        // the process of merging should fail inside ProfileBuilder on the first
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

            if (Options_.KeepProcesses) {
                builder.SetThread(MapThread(key.GetThread()));
            }

            builder.SetKernelStack(MapStack(key.GetKernelStack()));
            builder.SetUserStack(MapStack(key.GetUserStack()));

            for (i32 i = 0; i < key.GetLabelCount(); ++i) {
                TLabel label = key.GetLabel(i);
                if (!Options_.LabelFilter || Options_.LabelFilter(label)) {
                    builder.AddLabel(MapLabel(key.GetLabel(i)));
                }
            }

            return builder.Finish();
        });
    }

    TThreadId MapThread(TThread thread) {
        return Threads_.TryMap(thread.GetIndex(), [&thread, this] {
            auto builder = Builder_.AddThread();

            builder.SetThreadId(thread.GetThreadId());
            builder.SetProcessId(thread.GetProcessId());

            if (Options_.CleanupThreadNames) {
                TStringId sanitized = SanitizeThreadName(thread.GetThreadName());
                builder.SetThreadName(sanitized);
            } else {
                builder.SetThreadName(MapString(thread.GetThreadName()));
            }

            builder.SetProcessName(MapString(thread.GetProcessName()));

            for (i32 i = 0; i < thread.GetContainerCount(); ++i) {
                builder.AddContainerName(MapString(thread.GetContainer(i)));
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

            for (i32 i = 0; i < stack.GetStackFrameCount(); ++i) {
                builder.AddStackFrame(MapStackFrame(stack.GetStackFrame(i)));
            }

            return builder.Finish();
        });
    }

    TStackFrameId MapStackFrame(TStackFrame frame) {
        return StackFrames_.TryMap(frame.GetIndex(), [&, this] {
            auto builder = Builder_.AddStackFrame();

            if (Options_.KeepBinaries) {
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
            builder.SetPath(MapString(binary.GetPath()));

            return builder.Finish();
        });
    }

    TInlineChainId MapInlineChain(TInlineChain chain) {
        return InlineChains_.TryMap(chain.GetIndex(), [&, this] {
            auto builder = Builder_.AddInlineChain();

            for (i32 i = 0; i < chain.GetLineCount(); ++i) {
                auto line = chain.GetLine(i);

                auto lineBuilder = builder.AddLine();
                if (Options_.KeepLineNumbers) {
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
    const TMergeOptions Options_;
    const TProfile Profile_;
    const bool IsFirstProfile_;

    TIndexRemapping<TStringId> Strings_;
    TIndexRemapping<TValueTypeId> ValueTypes_;
    TIndexRemapping<TSampleId> Samples_;
    TIndexRemapping<TSampleKeyId> SampleKeys_;
    TIndexRemapping<TStackId> Stacks_;
    TIndexRemapping<TBinaryId> Binaries_;
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
    TImpl(NProto::NProfile::Profile* merged, TMergeOptions options)
        : Options_{options}
        , Builder_{merged}
    {}

    void Finish() {
        Builder_.Finish();
    }

    void Add(const NProto::NProfile::Profile& proto) {
        TProfile profile{&proto};
        TSingleProfileMerger{Builder_, Options_, profile, ProfileCount_++}.Merge();
    }

private:
    const TMergeOptions Options_;
    TProfileBuilder Builder_;
    ui32 ProfileCount_ = 0;
};

////////////////////////////////////////////////////////////////////////////////

TProfileMerger::TProfileMerger(NProto::NProfile::Profile* merged, TMergeOptions options)
    : Impl_{MakeHolder<TImpl>(merged, options)}
{}

TProfileMerger::~TProfileMerger() = default;

void TProfileMerger::Add(const NProto::NProfile::Profile& proto) {
    return Impl_->Add(proto);
}

void TProfileMerger::Finish() {
    return Impl_->Finish();
}

////////////////////////////////////////////////////////////////////////////////

void MergeProfiles(
    TConstArrayRef<NProto::NProfile::Profile> profiles,
    NProto::NProfile::Profile* merged,
    TMergeOptions options
) {
    TProfileMerger merger{merged, options};
    for (auto& profile : profiles) {
        merger.Add(profile);
    }
    merger.Finish();
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NProfile
