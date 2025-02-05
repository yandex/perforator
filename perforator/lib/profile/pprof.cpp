#include "builder.h"
#include "compact_map.h"
#include "pprof.h"
#include "profile.h"

#include <perforator/lib/permutation/permutation.h>

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>
#include <library/cpp/containers/absl_flat_hash/flat_hash_set.h>
#include <library/cpp/iterator/enumerate.h>
#include <library/cpp/iterator/zip.h>
#include <library/cpp/yt/compact_containers/compact_vector.h>

#include <util/digest/city.h>
#include <util/digest/multi.h>
#include <util/generic/bitops.h>
#include <util/generic/function_ref.h>
#include <util/generic/hash_set.h>
#include <util/generic/maybe.h>
#include <util/generic/size_literals.h>
#include <util/generic/typetraits.h>
#include <util/stream/format.h>
#include <util/system/yassert.h>


namespace NPerforator::NProfile {

namespace NDetail {

// Simple helper to prevent lossy implicit conversions.
// Profiles are represented as a bunch of integers of different bit width,
// and it is very error-prone to work with integeres in C++ when implicit
// conversions are everywhere. Moreover, Protobuf represents indices into
// repeated fields as `int`, and there is a lot of subtle bugs when combining
// protobuf structures with standard containers.
//
// For example, if there is a function `size_t Foo();`, one can write
// `int result = Foo()`, potentially lossing precision. To solve this, use
// `TExplicitReturnType<size_t> Foo();`.
template <typename T>
class TExplicitReturnType {
public:
    TExplicitReturnType(T value)
        : Value_{std::move(value)}
    {}

    template <std::same_as<T> U>
    operator U() const {
        return Value_;
    }

    template <typename U>
    U As() && {
        return static_cast<U>(std::move(Value_));
    }

private:
    T Value_;
};

class TStringTableConverter {
public:
    explicit TStringTableConverter(const NProto::NPProf::Profile& oldProfile, NProto::NProfile::StringTable* table)
        : Table_{table}
    {
        Populate(oldProfile);
    }

    void Populate(const NProto::NPProf::Profile& profile) {
        Y_ENSURE(profile.string_table(0).empty());

        TVector<size_t> permutation = MakeSortedPermutation(profile.string_table());
        Mapping_.resize(permutation.size());
        TString* strings = Table_->mutable_strings();

        for (auto i : permutation) {
            size_t offset = strings->size();
            ui32 id = Table_->offset_size();

            const TString& str = profile.string_table(i);
            strings->append(str);
            Table_->add_offset(offset);
            Table_->add_length(str.size());
            Mapping_[i] = id;
        }

        Validate(profile);
    }

    ui32 Remap(i64 id) const {
        Y_ENSURE(static_cast<size_t>(id) < Mapping_.size());
        return Mapping_[id];
    }

    void Validate(const NProto::NPProf::Profile& profile) const {
        for (size_t i = 0; i < Mapping_.size(); ++i) {
            size_t id = Remap(i);
            TStringBuf buf{Table_->strings().data() + Table_->offset(id), Table_->length(id)};
            Y_ENSURE(profile.string_table(i) == buf);
        }
    }

private:
    TVector<ui32> Mapping_;
    NProto::NProfile::StringTable* Table_;
};

template <CStrongIndex Index>
class TIndexedEntityRemapping {
public:
    struct TRemappedIndex {
        size_t OldPosition = 0;
        Index NewIndex = Index::Invalid();

        bool operator==(const TRemappedIndex& rhs) const = default;
    };

public:
    explicit TIndexedEntityRemapping(size_t sizeHint)
        : Mapping_{Max<size_t>(sizeHint + 10, 1024)}
    {
        Add(0ul, Max<size_t>(), Index::Zero());
    }

    bool IsEmpty() const {
        // Empty remappings contains exactly one zero value.
        return Mapping_.Size() == 1;
    }

    void Add(TExplicitType<ui64> oldIndex, TExplicitType<size_t> oldPosition, Index newIndex) {
        // Protobuf message size must not exceed 2GiB,
        // so indices into repeated fields must fit into signed 32-bit number.
        // We abuse this knowledge to reduce size of parsed profile in memory.
        Y_ENSURE(oldPosition < Max<i32>() || oldIndex == 0);
        Y_ENSURE(newIndex.IsValid());

        Y_ENSURE(Mapping_.TryEmplace(oldIndex, TRemappedIndex{
            .OldPosition = oldPosition,
            .NewIndex = newIndex,
        }), "Duplicate id " << oldIndex.Value());
    }

    TExplicitReturnType<size_t> GetOldPosition(ui64 oldIndex) const {
        return Mapping_.At(oldIndex).OldPosition;
    }

    Index GetNewIndex(ui64 oldIndex) const {
        return Mapping_.At(oldIndex).NewIndex;
    }

    TRemappedIndex GetPosition(ui64 oldIndex) const {
        return Mapping_.At(oldIndex);
    }

private:
    TCompactIntegerMap<ui64, TRemappedIndex> Mapping_;
};

class TThreadInfoParser {
public:
    struct TLabelKeys {
        static constexpr TStringBuf ThreadId = "tid";
        static constexpr TStringBuf ProcessId = "pid";
        static constexpr TStringBuf ProcessName = "process_comm";
        static constexpr TStringBuf ThreadName = "thread_comm";
        static constexpr TStringBuf ThreadNameDeprecated = "comm";
        static constexpr TStringBuf WorkloadName = "workload";

        static inline THashSet<TStringBuf> AllKeys{
            ThreadId,
            ProcessId,
            ProcessName,
            ThreadName,
            ThreadNameDeprecated,
            WorkloadName,
        };
    };

public:
    TThreadInfoParser(const NProto::NPProf::Profile* profile, std::function<TStringId(i64)> strtab)
        : Profile_{profile}
        , MapString_{strtab}
    {
        Y_ABORT_UNLESS(profile && strtab);
    }

    TThreadInfoParser(const NProto::NPProf::Profile* profile, const TStringTableConverter* strtab)
        : TThreadInfoParser{profile, [strtab](i64 id) {
            return TStringId::FromInternalIndex(strtab->Remap(id));
        }}
    {}

    bool Consume(const NProto::NPProf::Label& label) {
        TStringBuf key = Profile_->string_table(label.key());

        if (key == TLabelKeys::ThreadId) {
            Info_.ThreadId = label.num();
            return true;
        }

        if (key == TLabelKeys::ProcessId) {
            Info_.ProcessId = label.num();
            return true;
        }

        if (key == TLabelKeys::ProcessName) {
            Info_.ProcessNameIdx = MapString_(label.str());
            return true;
        }

        if (key == TLabelKeys::ThreadName) {
            Info_.ThreadNameIdx = MapString_(label.str());
            return true;
        } else if (key == TLabelKeys::ThreadNameDeprecated) {
            Info_.ThreadNameIdx = MapString_(label.str());
            return true;
        }

        if (key == TLabelKeys::WorkloadName) {
            Info_.ContainerIdx.push_back(MapString_(label.str()));
            return true;
        }

        return false;
    }

    TThreadInfo Finish() && {
        return std::move(Info_);
    }

private:
    const NProto::NPProf::Profile* Profile_ = nullptr;
    std::function<TStringId(i64)> MapString_;
    TThreadInfo Info_;
};

class TConverterContext {
    static constexpr TStringBuf KernelSpecialMapping{"[kernel]"};
    static constexpr TStringBuf PythonSpecialMapping{"[python]"};

    enum class ESpecialMappingKind {
        None,
        Kernel,
        Python,
    };

public:
    explicit TConverterContext(const NProto::NPProf::Profile& from, NProto::NProfile::Profile* to)
        : OldProfile_{from}
        , Builder_{to}
        , BinaryMapping_{static_cast<size_t>(OldProfile_.mapping_size())}
        , FunctionMapping_{static_cast<size_t>(OldProfile_.function_size())}
        , LocationMapping_{static_cast<size_t>(OldProfile_.location_size())}
    {}

    void Convert() && {
        ConvertStrings();
        ConvertBinaries();
        ConvertFunctions();
        ConvertLocations();
        ConvertComments();
        ConvertSamples();
    }

private:
    void ConvertStrings() {
        Y_ENSURE(OldProfile_.string_table_size() > 0);
        Y_ENSURE(OldProfile_.string_table(0) == "");

        // Sort strings to make strtab more compression-friendly.
        // Probably this should be done by the builder.
        TVector<size_t> permutation = MakeSortedPermutation(OldProfile_.string_table());
        for (size_t i : permutation) {
            TStringBuf string = OldProfile_.string_table(i);
            Strings_.TryEmplace(i, Builder_.AddString(string));
        }
    }

    void ConvertBinaries() {
        Y_ABORT_UNLESS(BinaryMapping_.IsEmpty());

        TMaybe<ui64> oldKernelMappingId;
        TMaybe<ui64> oldPythonMappingId;
        for (auto&& [i, mapping] : Enumerate(OldProfile_.mapping())) {
            Y_ENSURE(mapping.id() != 0, "Mapping id should be nonzero");

            auto builder = Builder_.AddBinary();
            builder.SetBuildId(ConvertString(mapping.build_id()));
            builder.SetPath(ConvertString(mapping.filename()));
            BinaryMapping_.Add(mapping.id(), i, builder.Finish());

            if (OldProfile_.string_table(mapping.filename()) == KernelSpecialMapping) {
                Y_ENSURE(!oldKernelMappingId, "Found more than one kernel mapping");
                oldKernelMappingId = mapping.id();
            }
            if (OldProfile_.string_table(mapping.filename()) == PythonSpecialMapping) {
                Y_ENSURE(!oldPythonMappingId, "Found more than one python mapping");
                oldPythonMappingId = mapping.id();
            }
        }

        OldKernelMappingId_ = oldKernelMappingId.GetOrElse(Max<ui64>());
        OldPythonMappingId_ = oldPythonMappingId.GetOrElse(Max<ui64>());
    }

    void ConvertFunctions() {
        Y_ABORT_UNLESS(FunctionMapping_.IsEmpty());

        for (auto&& [i, function] : Enumerate(OldProfile_.function())) {
            Y_ENSURE(function.id() != 0, "Function id should be nonzero");

            auto builder = Builder_.AddFunction();
            builder.SetName(ConvertString(function.name()));
            builder.SetSystemName(ConvertString(function.system_name()));
            builder.SetFileName(ConvertString(function.filename()));
            builder.SetStartLine(function.start_line());
            FunctionMapping_.Add(function.id(), i, builder.Finish());
        }
    }

    void ConvertLocations() {
        Y_ABORT_UNLESS(LocationMapping_.IsEmpty());

        for (auto&& [i, location] : Enumerate(OldProfile_.location())) {
            Y_ENSURE(location.id() != 0, "Location id should be nonzero");

            auto frame = Builder_.AddStackFrame();
            if (location.mapping_id()) {
                auto [mappingId, binaryId] = BinaryMapping_.GetPosition(location.mapping_id());
                auto&& mapping = OldProfile_.mapping(mappingId);
                i64 binaryOffset = location.address() + (i64)mapping.file_offset() - (i64)mapping.memory_start();

                frame.SetBinary(binaryId);
                frame.SetBinaryOffset(binaryOffset);
            }

            auto chain = Builder_.AddInlineChain();
            for (auto&& line : location.line()) {
                chain
                    .AddLine()
                    .SetLine(line.line())
                    .SetColumn(line.column())
                    .SetFunction(FunctionMapping_.GetNewIndex(line.function_id()))
                    .Finish();
            }
            frame.SetInlineChain(chain.Finish());

            LocationMapping_.Add(location.id(), i, frame.Finish());

            switch (ClassifySpecialMapping(location.id())) {
            case ESpecialMappingKind::None:
                break;

            case ESpecialMappingKind::Kernel:
                OldKernelLocationIds_.Insert(location.id());
                break;

            case ESpecialMappingKind::Python:
                OldPythonLocationIds_.Insert(location.id());
                break;
            }
        }
    }

    void ConvertSamples() {
        ConvertSampleTypes();
        for (auto&& sample : OldProfile_.sample()) {
            ConvertSample(sample);
        }
    }

    void ConvertSampleTypes() {
        Y_ABORT_UNLESS(ValueTypes_.empty());
        for (auto&& value : OldProfile_.sample_type()) {
            auto id = Builder_.AddValueType(
                ConvertString(value.type()),
                ConvertString(value.unit())
            );
            ValueTypes_.push_back(id);
        }
    }

    void ConvertSample(const NProto::NPProf::Sample& sample) {
        auto keyBuilder = Builder_.AddSampleKey();
        ConvertSampleStack(keyBuilder, sample);
        ConvertSampleLabels(keyBuilder, sample);

        auto sampleBuilder = Builder_.AddSample();
        sampleBuilder.SetSampleKey(keyBuilder.Finish());
        ConvertSampleValues(sampleBuilder, sample);
        sampleBuilder.Finish();
    }

    void ConvertSampleStack(TProfileBuilder::TSampleKeyBuilder& builder, const NProto::NPProf::Sample& sample) {
        auto kstack = Builder_.AddStack();
        auto ustack = Builder_.AddStack();

        bool insideKernel = true;
        for (ui64 location : sample.location_id()) {
            auto frame = LocationMapping_.GetNewIndex(location);

            if (OldPythonLocationIds_.Contains(location)) {
                continue;
            }

            if (OldKernelLocationIds_.Contains(location)) {
                Y_ENSURE(insideKernel, "Unexpected mixed userspace & kernelspace stack");
                kstack.AddStackFrame(frame);
            } else {
                insideKernel = false;
                ustack.AddStackFrame(frame);
            }
        }

        builder.SetKernelStack(kstack.Finish());
        builder.SetUserStack(ustack.Finish());
    }

    ESpecialMappingKind ClassifySpecialMapping(ui64 oldLocation) const {
        ui64 oldPosition = LocationMapping_.GetOldPosition(oldLocation);
        ui64 mappingId = OldProfile_.location(oldPosition).mapping_id();

        if (mappingId == OldKernelMappingId_) {
            return ESpecialMappingKind::Kernel;
        } else if (mappingId == OldPythonMappingId_) {
            return ESpecialMappingKind::Python;
        } else {
            return ESpecialMappingKind::None;
        }
    }

    void ConvertSampleValues(TProfileBuilder::TSampleBuilder& builder, const NProto::NPProf::Sample& sample) {
        for (auto&& [i, value] : Enumerate(sample.value())) {
            builder.AddValue(ValueTypes_.at(i), value);
        }
    }

    void ConvertSampleLabels(TProfileBuilder::TSampleKeyBuilder& keyBuilder, const NProto::NPProf::Sample& sample) {
        NDetail::TThreadInfoParser tip{&OldProfile_, [this](i64 id){
            return ConvertString(id);
        }};

        for (auto&& label : sample.label()) {
            if (tip.Consume(label)) {
                continue;
            }

            TLabelId id = TLabelId::Invalid();
            if (0 != label.num()) {
                id = Builder_.AddNumericLabel(ConvertString(label.key()), label.num());
            } else {
                id = Builder_.AddStringLabel(ConvertString(label.key()), ConvertString(label.str()));
            }
            keyBuilder.AddLabel(id);
        }

        TThreadId thread = Builder_.AddThread(std::move(tip).Finish());
        keyBuilder.SetThread(thread);
    }

    void ConvertComments() {
        for (i64 comment : OldProfile_.comment()) {
            Builder_.AddComment(ConvertString(comment));
        }
    }

private:
    TStringId ConvertString(i64 id) const {
        return Strings_.At(id);
    }

private:
    const NProto::NPProf::Profile& OldProfile_;
    TProfileBuilder Builder_;

    TCompactIntegerMap<ui32, TStringId> Strings_;
    NDetail::TIndexedEntityRemapping<TBinaryId> BinaryMapping_;
    NDetail::TIndexedEntityRemapping<TFunctionId> FunctionMapping_;
    NDetail::TIndexedEntityRemapping<TStackFrameId> LocationMapping_;
    TVector<TValueTypeId> ValueTypes_;
    TCompactIntegerSet<ui64> OldKernelLocationIds_;
    TCompactIntegerSet<ui64> OldPythonLocationIds_;
    ui64 OldKernelMappingId_ = Max<ui64>();
    ui64 OldPythonMappingId_ = Max<ui64>();
};

////////////////////////////////////////////////////////////////////////////////

class TOldProfileConverter {
public:
    explicit TOldProfileConverter(
        const NProto::NProfile::Profile& newProfile,
        NProto::NPProf::Profile* oldProfile
    )
        : SourceProfile_{&newProfile}
        , OldProfile_{*oldProfile}
    {}

    void Convert() && {
        ConvertStringTable();
        ConvertValueTypes();
        ConvertComments();
        ConvertMappings();
        ConvertFunctions();
        ConvertLocations();
        ConvertSamples();
    }

private:
    void ConvertStringTable() {
        for (auto str : SourceProfile_.Strings()) {
            TStringBuf view = str.View();

            // We need to collect ids of some well-known strings, for example, process info label keys.
            if (IsWellKnownString(view)) {
                WellKnownStringIds_.try_emplace(view, OldProfile_.string_table_size());
            }

            OldProfile_.add_string_table(view);
        }

        Y_ENSURE(OldProfile_.string_table_size() > 0);
        Y_ENSURE(OldProfile_.string_table(0).empty());
    }

    bool IsWellKnownString(TStringBuf str) const {
        return TThreadInfoParser::TLabelKeys::AllKeys.contains(str);
    }

    int GetStringIndex(TStringBuf key) {
        if (auto ptr = WellKnownStringIds_.FindPtr(key)) {
            return *ptr;
        }

        int id = OldProfile_.string_table_size();
        WellKnownStringIds_[key] = id;
        *OldProfile_.add_string_table() = key;
        return id;
    }

    void ConvertValueTypes() {
        for (TValueType valueType : SourceProfile_.ValueTypes()) {
            NProto::NPProf::ValueType* type = OldProfile_.add_sample_type();
            type->set_type(*valueType.GetType().GetIndex());
            type->set_unit(*valueType.GetUnit().GetIndex());
        }
    }

    void ConvertComments() {
        for (TComment comment : SourceProfile_.Comments()) {
            OldProfile_.add_comment(*comment.GetString().GetIndex());
        }
    }

    void ConvertMappings() {
        const auto binaries = SourceProfile_.Binaries();

        OldProfile_.mutable_mapping()->Reserve(binaries.GetApproxSize());
        for (auto [i, binary] : Enumerate(binaries)) {
            if (i == 0) {
                // First binary is empty ant should not be present in pprof.
                continue;
            }

            NProto::NPProf::Mapping* mapping = OldProfile_.add_mapping();
            mapping->set_id(i);
            mapping->set_build_id(*binary.GetBuildId().GetIndex());
            mapping->set_filename(*binary.GetPath().GetIndex());

            // Our new profile represntation is lossy.
            // We do not know exact addresess of mappings.
            static constexpr ui64 fakeMappingSize = 128_GB;
            mapping->set_memory_start(i * fakeMappingSize);
            mapping->set_memory_limit((i + 1) * fakeMappingSize);
            mapping->set_file_offset(0);
        }
    }

    void ConvertFunctions() {
        const auto functions = SourceProfile_.Functions();

        OldProfile_.mutable_function()->Reserve(functions.GetApproxSize());
        for (auto [i, func] : Enumerate(functions)) {
            // Skip first function which must be empty.
            if (i == 0) {
                continue;
            }

            NProto::NPProf::Function* function = OldProfile_.add_function();
            function->set_id(i);
            function->set_name(*func.GetName().GetIndex());
            function->set_system_name(*func.GetSystemName().GetIndex());
            function->set_filename(*func.GetFileName().GetIndex());
            function->set_start_line(func.GetStartLine());
        }
    }

    void ConvertLocations() {
        const auto frames = SourceProfile_.StackFrames();

        // We add first null location as the "unknown" location and shift location ids by one.
        // pprof expects that Profile.sample.location_id are non-zero.
        OldProfile_.mutable_location()->Reserve(frames.GetApproxSize());
        for (auto [i, frame] : Enumerate(frames)) {
            NProto::NPProf::Location* location = OldProfile_.add_location();
            location->set_id(i + 1);

            auto inlineChain = frame.GetInlineChain();
            for (i32 i = 0; i < inlineChain.GetLineCount(); ++i) {
                auto sourceLine = inlineChain.GetLine(i);

                NProto::NPProf::Line* line = location->add_line();
                line->set_function_id(*sourceLine.GetFunction().GetIndex());
                line->set_line(sourceLine.GetLine());
                line->set_column(sourceLine.GetColumn());
            }

            ui32 binaryId = *frame.GetBinary().GetIndex();
            i64 binaryOffset = frame.GetBinaryOffset();;
            if (binaryId == 0) {
                Y_ENSURE(binaryOffset == 0, "Malformed profile");
                location->set_mapping_id(0);
                location->set_address(0);
            } else {
                const NProto::NPProf::Mapping& mapping = OldProfile_.mapping(binaryId - 1);

                // We need to build artificial address value.
                // See symmetric conversion in TConverterContext::ConvertLocations.
                i64 address = binaryOffset - (i64)mapping.file_offset() + (i64)mapping.memory_start();
                Y_ENSURE(address > 0);

                location->set_mapping_id(binaryId);
                location->set_address(address);
            }
        }
    }

    void ConvertSamples() {
        const auto samples = SourceProfile_.Samples();
        // const NProto::NProfile::SampleKeys& keys = NewProfile_.sample_keys();

        for (TSample newSample : samples) {
            NProto::NPProf::Sample* oldSample = OldProfile_.add_sample();

            // Fill Sample.value
            for (i32 i = 0; i < newSample.GetValueCount(); ++i) {
                oldSample->add_value(newSample.GetValue(i));
            }

            // Fill Sample.stack
            ConvertSampleStack(oldSample, newSample.GetKey().GetKernelStack());
            ConvertSampleStack(oldSample, newSample.GetKey().GetUserStack());

            // Fill Sample.labels
            ConvertSampleThreadInfo(oldSample, newSample.GetKey().GetThread());
            ConvertSampleLabels(oldSample, newSample.GetKey());
        }
    }

    void ConvertSampleStack(NProto::NPProf::Sample* sample, TStack stack) {
        for (i32 i = 0; i < stack.GetStackFrameCount(); ++i) {
            // We shift location ids by 1 because pprof does not support zero location ids.
            // See corresponding comment inside ConvertLocations.
            sample->add_location_id(*stack.GetStackFrame(i).GetIndex() + 1);
        }
    }

    void ConvertSampleThreadInfo(NProto::NPProf::Sample* sample, TThread thread) {
        using TKeys = TThreadInfoParser::TLabelKeys;

        if (auto pid = thread.GetProcessId()) {
            AddNumberLabel(sample, TKeys::ProcessId, pid);
        }
        if (auto tid = thread.GetThreadId()) {
            AddNumberLabel(sample, TKeys::ThreadId, tid);
        }
        if (auto name = thread.GetProcessName()) {
            AddStringIdxLabel(sample, TKeys::ProcessName, name);
        }
        if (auto name = thread.GetThreadName()) {
            AddStringIdxLabel(sample, TKeys::ThreadName, name);
        }
        for (i32 i = 0; i < thread.GetContainerCount(); ++i) {
            AddStringIdxLabel(sample, TKeys::WorkloadName, thread.GetContainer(i));
        }
    }

    void ConvertSampleLabels(NProto::NPProf::Sample* sample, TSampleKey key) {
        for (i32 i = 0; i < key.GetLabelCount(); ++i) {
            TLabel newLabel = key.GetLabel(i);

            auto* label = sample->add_label();
            if (newLabel.IsNumber()) {
                label->set_key(*newLabel.GetKey().GetIndex());
                label->set_num(newLabel.GetNumber());
            } else {
                label->set_key(*newLabel.GetKey().GetIndex());
                label->set_str(*newLabel.GetString().GetIndex());
            }
        }
    }

    NProto::NPProf::Label* AddLabel(NProto::NPProf::Sample* sample, TStringBuf key) {
        auto* label = sample->add_label();
        label->set_key(GetStringIndex(key));
        return label;
    }

    void AddNumberLabel(NProto::NPProf::Sample* sample, TStringBuf key, i64 value) {
        AddLabel(sample, key)->set_num(value);
    }

    void AddStringIdxLabel(NProto::NPProf::Sample* sample, TStringBuf key, TStringRef valueIdx) {
        AddLabel(sample, key)->set_str(*valueIdx.GetIndex());
    }

private:
    const NProfile::TProfile SourceProfile_;
    NProto::NPProf::Profile& OldProfile_;
    THashMap<TString, int> WellKnownStringIds_;
};

} // namespace NDetail

void ConvertFromPProf(const NProto::NPProf::Profile& from, NProto::NProfile::Profile* to) {
    Y_ABORT_UNLESS(to, "Expected non-null output pointer");
    NDetail::TConverterContext{from, to}.Convert();
}

void ConvertToPProf(const NProto::NProfile::Profile& from, NProto::NPProf::Profile* to) {
    Y_ABORT_UNLESS(to, "Expected non-null output pointer");
    NDetail::TOldProfileConverter{from, to}.Convert();
}

} // namespace NPerforator::NProfile
