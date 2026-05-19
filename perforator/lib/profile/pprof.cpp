#include "builder.h"
#include "compact_map.h"
#include "pprof.h"
#include "profile.h"

#include <google/protobuf/io/coded_stream.h>
#include <google/protobuf/wire_format_lite.h>

#include <perforator/lib/permutation/permutation.h>

#include <library/cpp/containers/absl_flat_hash/flat_hash_map.h>
#include <library/cpp/containers/absl_flat_hash/flat_hash_set.h>
#include <library/cpp/iterator/enumerate.h>
#include <library/cpp/iterator/zip.h>
#include <library/cpp/protobuf/inplace/inplace.h>

#include <util/digest/city.h>
#include <util/digest/multi.h>
#include <util/generic/bitops.h>
#include <util/generic/cast.h>
#include <util/generic/function_ref.h>
#include <util/generic/hash_set.h>
#include <util/generic/maybe.h>
#include <util/generic/size_literals.h>
#include <util/generic/typetraits.h>
#include <util/stream/format.h>
#include <util/stream/mem.h>
#include <util/stream/zlib.h>
#include <util/system/yassert.h>


namespace NPerforator::NProfile {

namespace NDetail {

static constexpr TStringBuf KernelSpecialMapping{"[kernel]"};
static constexpr TStringBuf PythonSpecialMapping{"[python]"};

// Simple helper to prevent lossy implicit conversions.
// Profiles are represented as a bunch of integers of different bit width,
// and it is very error-prone to work with integers in C++ when implicit
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

template <CStrongIndex Index>
class TIndexedEntityRemapping {
public:
    struct TRemappedIndex {
        ui32 OldPosition = 0;
        Index NewIndex = Index::Invalid();

        bool operator==(const TRemappedIndex& rhs) const = default;
    };

public:
    explicit TIndexedEntityRemapping(size_t sizeHint)
        : Mapping_{Max<size_t>(sizeHint + 10, 1024)}
    {
        Add((ui64)0ul, Max<ui32>(), Index::Zero());
    }

    bool IsEmpty() const {
        // Empty remappings contains exactly one zero value.
        return Mapping_.Size() == 1;
    }

    void Add(TExplicitType<ui64> oldIndex, TExplicitType<ui32> oldPosition, Index newIndex) {
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

// We could parse pprof data from a stream without buffering, but that would require a specific serialization order
// to handle field dependencies. However, the order produced by the standard pprof tooling is not suitable for this.
// An alternative is to build the profile using the same IDs, which would bypass the builder's deduplication logic.
// For now, we are taking the simpler approach of parsing it from a contiguous block of memory.
class TFromPProfBytesConverterContext {
public:
    explicit TFromPProfBytesConverterContext(TStringBuf from Y_LIFETIME_BOUND, NProto::NProfile::Profile* to)
        : From_(from)
        , Builder_(to)
    {}

    void Convert() && {
        ParseProfile(From_);
    }

private:
    void ParseSampleType(NInPlaceProto::TRegionParser& parent) {
        using NProto::NPProf::ValueType;
        NInPlaceProto::TRegionParser parser{NInPlaceProto::AsSerialized<ValueType>(parent.GetBytesAsBuf())};

        i64 type = 0;
        i64 unit = 0;

        while (ui32 fieldNumber = parser.NextFieldNumber()) {
            switch (fieldNumber) {
                case ValueType::kTypeFieldNumber:
                    type = parser.GetInt64();
                    break;
                case ValueType::kUnitFieldNumber:
                    unit = parser.GetInt64();
                    break;
                default:
                    parser.SkipField();
            }
        }

        Y_ENSURE(!parser.IsCorrupted());

        ValueTypeMapping_.push_back(Builder_.AddValueType(StringMapping_.at(type), StringMapping_.at(unit)));
    }

    void ParseSampleLocationId(NInPlaceProto::TRegionParser& parent, TProfileBuilder::TSimpleSampleKeyBuilder& keyBuilder) {
        auto handle = [&](ui64 value) {
            TStackFrameId frame = LocationMapping_->GetNewIndex(value);
            keyBuilder.AddFrame(frame);
        };

        if (parent.GetWireType() == google::protobuf::internal::WireFormatLite::WIRETYPE_LENGTH_DELIMITED) {
            NInPlaceProto::TRegionDataProvider provider{NInPlaceProto::AsSerialized<void>(parent.GetBytesAsBuf())};
            while (provider.NotEmpty()) {
                handle((ui64)provider.ReadVarint64());
            }
            Y_ENSURE(!provider.IsCorrupted());
        } else {
            handle(parent.GetUInt64());
        }
    }

    void ParseSampleValue(NInPlaceProto::TRegionParser& parent, TProfileBuilder::TSampleBuilder& builder, size_t& valueIndex) {
        auto handle = [&](i64 value) {
            builder.AddValue(ValueTypeMapping_.at(valueIndex++), value);
        };

        if (parent.GetWireType() == google::protobuf::internal::WireFormatLite::WIRETYPE_LENGTH_DELIMITED) {
            NInPlaceProto::TRegionDataProvider provider{NInPlaceProto::AsSerialized<void>(parent.GetBytesAsBuf())};
            while (provider.NotEmpty()) {
                handle((i64)provider.ReadVarint64());
            }
            Y_ENSURE(!provider.IsCorrupted());
        } else {
            handle(parent.GetInt64());
        }
    }

    void ParseSampleLabel(NInPlaceProto::TRegionParser& parent, TProfileBuilder::TSimpleSampleKeyBuilder& keyBuilder) {
        using NProto::NPProf::Label;
        NInPlaceProto::TRegionParser parser{NInPlaceProto::AsSerialized<Label>(parent.GetBytesAsBuf())};

        i64 key = 0;
        i64 str = 0;
        i64 num = 0;

        while (ui32 fieldNumber = parser.NextFieldNumber()) {
            switch (fieldNumber) {
                case Label::kKeyFieldNumber:
                    key = parser.GetInt64();
                    break;
                case Label::kStrFieldNumber:
                    str = parser.GetInt64();
                    break;
                case Label::kNumFieldNumber:
                    num = parser.GetInt64();
                    break;
                default:
                    parser.SkipField();
            }
        }

        Y_ENSURE(!parser.IsCorrupted());

        auto keyId = StringMapping_.at(key);

        TLabelId id = TLabelId::Invalid();
        // TODO(ayles): Shouldn't this be reversed?
        if (num != 0) {
            id = Builder_.AddNumericLabel(keyId, num);
        } else {
            id = Builder_.AddStringLabel(keyId, StringMapping_.at(str));
        }

        keyBuilder.AddLabel(id);
    }

    void ParseSample(NInPlaceProto::TRegionParser& parent) {
        using NProto::NPProf::Sample;
        NInPlaceProto::TRegionParser parser{NInPlaceProto::AsSerialized<Sample>(parent.GetBytesAsBuf())};

        auto keyBuilder = Builder_.AddSimpleSampleKey();
        auto sampleBuilder = Builder_.AddSample();

        size_t valueIndex = 0;

        while (ui32 fieldNumber = parser.NextFieldNumber()) {
            switch (fieldNumber) {
                case Sample::kLocationIdFieldNumber:
                    ParseSampleLocationId(parser, keyBuilder);
                    break;
                case Sample::kValueFieldNumber:
                    ParseSampleValue(parser, sampleBuilder, valueIndex);
                    break;
                case Sample::kLabelFieldNumber:
                    ParseSampleLabel(parser, keyBuilder);
                    break;
                default:
                    parser.SkipField();
            }
        }

        Y_ENSURE(!parser.IsCorrupted());

        sampleBuilder.SetSampleKey(keyBuilder.Finish());
        sampleBuilder.Finish();
    }

    void ParseStringTable(NInPlaceProto::TRegionParser& parent) {
        auto str = parent.GetStringAsBuf();
        TStringId id = Builder_.AddString(str);

        StringMapping_.push_back(id);
    }

    void ParseMapping(NInPlaceProto::TRegionParser& parent) {
        using NProto::NPProf::Mapping;
        NInPlaceProto::TRegionParser parser{NInPlaceProto::AsSerialized<Mapping>(parent.GetBytesAsBuf())};

        auto builder = Builder_.AddBinary();
        ui64 id = 0;
        ui64 memoryStart = 0;
        ui64 fileOffset = 0;
        TStringId path = TStringId::Zero();
        TStringId buildId = TStringId::Zero();

        while (ui32 fieldNumber = parser.NextFieldNumber()) {
            switch (fieldNumber) {
                case Mapping::kIdFieldNumber:
                    id = parser.GetUInt64();
                    break;
                case Mapping::kMemoryStartFieldNumber:
                    memoryStart = parser.GetUInt64();
                    break;
                case Mapping::kFileOffsetFieldNumber:
                    fileOffset = parser.GetUInt64();
                    break;
                case Mapping::kFilenameFieldNumber:
                    path = StringMapping_.at(parser.GetInt64());
                    break;
                case Mapping::kBuildIdFieldNumber:
                    buildId = StringMapping_.at(parser.GetInt64());
                    break;
                default:
                    parser.SkipField();
            }
        }

        builder.SetPath(path);
        builder.SetBuildId(buildId);

        Y_ENSURE(!parser.IsCorrupted());

        Y_ENSURE(id != 0, "Mapping id should be nonzero");

        auto binaryId = builder.Finish();
        BinaryMapping_->Add(id, (ui32)0, binaryId);
        AddressAdjustmentMap_->EmplaceUnique(id, fileOffset - memoryStart);
    }

    void ParseFunction(NInPlaceProto::TRegionParser& parent) {
        using NProto::NPProf::Function;
        NInPlaceProto::TRegionParser parser{NInPlaceProto::AsSerialized<Function>(parent.GetBytesAsBuf())};

        auto builder = Builder_.AddFunction();
        ui64 id = 0;

        while (ui32 fieldNumber = parser.NextFieldNumber()) {
            switch (fieldNumber) {
                case Function::kIdFieldNumber:
                    id = parser.GetUInt64();
                    break;
                case Function::kNameFieldNumber:
                    builder.SetName(StringMapping_.at(parser.GetInt64()));
                    break;
                case Function::kSystemNameFieldNumber:
                    builder.SetSystemName(StringMapping_.at(parser.GetInt64()));
                    break;
                case Function::kFilenameFieldNumber:
                    builder.SetFileName(StringMapping_.at(parser.GetInt64()));
                    break;
                case Function::kStartLineFieldNumber:
                    builder.SetStartLine(parser.GetInt64());
                    break;
                default:
                    parser.SkipField();
            }
        }

        Y_ENSURE(!parser.IsCorrupted());

        Y_ENSURE(id != 0, "Function id should be nonzero");

        FunctionMapping_->Add(id, (ui32)0, builder.Finish());
    }

    void ParseLine(NInPlaceProto::TRegionParser& parent, TProfileBuilder::TInlineChainBuilder& inlineChainBuilder) {
        using NProto::NPProf::Line;
        NInPlaceProto::TRegionParser parser{NInPlaceProto::AsSerialized<Line>(parent.GetBytesAsBuf())};

        auto builder = inlineChainBuilder.AddLine();

        while (ui32 fieldNumber = parser.NextFieldNumber()) {
            switch (fieldNumber) {
                case Line::kFunctionIdFieldNumber:
                    builder.SetFunction(FunctionMapping_->GetNewIndex(parser.GetUInt64()));
                    break;
                case Line::kLineFieldNumber:
                    builder.SetLine(parser.GetInt64());
                    break;
                case Line::kColumnFieldNumber:
                    builder.SetColumn(parser.GetInt64());
                    break;
                default:
                    parser.SkipField();
            }
        }

        Y_ENSURE(!parser.IsCorrupted());

        builder.Finish();
    }

    void ParseLocation(NInPlaceProto::TRegionParser& parent) {
        using NProto::NPProf::Location;
        NInPlaceProto::TRegionParser parser{NInPlaceProto::AsSerialized<Location>(parent.GetBytesAsBuf())};

        auto stackFrameBuilder = Builder_.AddStackFrame();
        auto inlineChainBuilder = Builder_.AddInlineChain();
        ui64 id = 0;
        ui64 mappingId = 0;
        ui64 address = 0;

        while (ui32 fieldNumber = parser.NextFieldNumber()) {
            switch (fieldNumber) {
                case Location::kIdFieldNumber:
                    id = parser.GetUInt64();
                    break;
                case Location::kMappingIdFieldNumber:
                    mappingId = parser.GetUInt64();
                    break;
                case Location::kAddressFieldNumber:
                    address = parser.GetUInt64();
                    break;
                case Location::kLineFieldNumber:
                    ParseLine(parser, inlineChainBuilder);
                    break;
                default:
                    parser.SkipField();
            }
        }

        Y_ENSURE(!parser.IsCorrupted());

        if (mappingId) {
            auto binaryId = BinaryMapping_->GetNewIndex(mappingId);
            stackFrameBuilder.SetBinary(binaryId);
            stackFrameBuilder.SetAddress(address + AddressAdjustmentMap_->At(mappingId));
        } else {
            stackFrameBuilder.SetAddress(address);
        }

        stackFrameBuilder.SetInlineChain(inlineChainBuilder.Finish());

        Y_ENSURE(id != 0, "Location id should be nonzero");

        LocationMapping_->Add(id, (ui32)0, stackFrameBuilder.Finish());
    }

    void ParseComment(NInPlaceProto::TRegionParser& parent) {
        auto handle = [&](i64 value) {
            Builder_.AddComment(StringMapping_.at(value));
        };

        if (parent.GetWireType() == google::protobuf::internal::WireFormatLite::WIRETYPE_LENGTH_DELIMITED) {
            NInPlaceProto::TRegionDataProvider provider{NInPlaceProto::AsSerialized<void>(parent.GetBytesAsBuf())};
            while (provider.NotEmpty()) {
                handle((i64)provider.ReadVarint64());
            }
            Y_ENSURE(!provider.IsCorrupted());
        } else {
            handle(parent.GetInt64());
        }
    }

    void ParseDefaultSampleType(NInPlaceProto::TRegionParser& parent) {
        i64 stringIndex = parent.GetInt64();
        if (stringIndex > 0) {
            TStringId stringId = StringMapping_.at(stringIndex);
            Builder_.Metadata().GetProto().set_default_sample_type(*stringId);
        }
    }

    void ParseDurationNanos(NInPlaceProto::TRegionParser& parent) {
        i64 duration = parent.GetInt64();
        Builder_.Metadata().GetProto().set_duration_nanoseconds(duration);
    }

    void ParseTimeNanos(NInPlaceProto::TRegionParser& parent) {
        i64 timeNanos = parent.GetInt64();
        auto* timestamp = Builder_.Metadata().GetProto().mutable_timestamp();
        timestamp->set_seconds(timeNanos / 1'000'000'000);
        timestamp->set_nanos(timeNanos % 1'000'000'000);
    }

    struct TCachedValue {
        const char* Start = nullptr;
        const char* End = nullptr;
        size_t Count = 0;
    };

    absl::flat_hash_map<ui32, TCachedValue> CacheFieldRangesAndCounts(TStringBuf buf) {
        absl::flat_hash_map<ui32, TCachedValue> ranges;

        NInPlaceProto::TRegionParser parser{buf.data(), buf.size()};

        const char* pos = (const char*)parser.GetCurrentPos();
        while (ui32 fieldNumber = parser.NextFieldNumber()) {
            parser.SkipField();

            const char* nextpos = (const char*)parser.GetCurrentPos();

            auto it = ranges.find(fieldNumber);
            if (it == ranges.end()) {
                ranges.try_emplace(it, fieldNumber, TCachedValue{pos, nextpos, 1});
            } else {
                it->second.End = nextpos;
                it->second.Count++;
            }

            pos = nextpos;
        }

        Y_ENSURE(!parser.IsCorrupted(), "Input is not a valid protobuf. "
            "Ensure the data is uncompressed pprof format.");

        return ranges;
    }

    void ParseProfile(TStringBuf buf) {
        using NProto::NPProf::Profile;

        auto parseField = [&](auto&& parser) {
            switch (parser.GetFieldNumber()) {
                case Profile::kSampleTypeFieldNumber:
                    ParseSampleType(parser);
                    break;
                case Profile::kSampleFieldNumber:
                    ParseSample(parser);
                    break;
                case Profile::kMappingFieldNumber:
                    ParseMapping(parser);
                    break;
                case Profile::kLocationFieldNumber:
                    ParseLocation(parser);
                    break;
                case Profile::kFunctionFieldNumber:
                    ParseFunction(parser);
                    break;
                case Profile::kStringTableFieldNumber:
                    ParseStringTable(parser);
                    break;
                case Profile::kCommentFieldNumber:
                    ParseComment(parser);
                    break;
                case Profile::kDefaultSampleTypeFieldNumber:
                    ParseDefaultSampleType(parser);
                    break;
                case Profile::kDurationNanosFieldNumber:
                    ParseDurationNanos(parser);
                    break;
                case Profile::kTimeNanosFieldNumber:
                    ParseTimeNanos(parser);
                    break;
                default:
                    parser.SkipField();
            }
        };

        auto ranges = CacheFieldRangesAndCounts(buf);

        StringMapping_.reserve(ranges[Profile::kStringTableFieldNumber].Count);
        ValueTypeMapping_.reserve(ranges[Profile::kSampleTypeFieldNumber].Count);

        AddressAdjustmentMap_.ConstructInPlace(ranges[Profile::kMappingFieldNumber].Count);
        BinaryMapping_.ConstructInPlace(ranges[Profile::kMappingFieldNumber].Count);
        FunctionMapping_.ConstructInPlace(ranges[Profile::kFunctionFieldNumber].Count);
        LocationMapping_.ConstructInPlace(ranges[Profile::kLocationFieldNumber].Count);

        // Fields should be processed in the specific order because of dependencies.
        for (auto&& desiredFieldNumber : {
            Profile::kStringTableFieldNumber,
            Profile::kDefaultSampleTypeFieldNumber,
            Profile::kDurationNanosFieldNumber,
            Profile::kTimeNanosFieldNumber,
            Profile::kMappingFieldNumber,
            Profile::kFunctionFieldNumber,
            Profile::kLocationFieldNumber,
            Profile::kCommentFieldNumber,
            Profile::kSampleTypeFieldNumber,
            Profile::kSampleFieldNumber,
        }) {
            auto range = ranges[desiredFieldNumber];
            NInPlaceProto::TRegionParser parser{range.Start, range.End};
            while (ui32 fieldNumber = parser.NextFieldNumber()) {
                if ((int)fieldNumber != desiredFieldNumber) {
                    parser.SkipField();
                } else {
                    parseField(parser);
                }
            }
            Y_ENSURE(!parser.IsCorrupted());
            ranges.erase(desiredFieldNumber);
        }
    }

private:
    TStringBuf From_;
    TProfileBuilder Builder_;
    TVector<TStringId> StringMapping_;
    TVector<TValueTypeId> ValueTypeMapping_;
    TMaybe<TCompactIntegerMap<ui64, ui64>> AddressAdjustmentMap_;
    TMaybe<NDetail::TIndexedEntityRemapping<TBinaryId>> BinaryMapping_;
    TMaybe<NDetail::TIndexedEntityRemapping<TFunctionId>> FunctionMapping_;
    TMaybe<NDetail::TIndexedEntityRemapping<TStackFrameId>> LocationMapping_;
};

using google::protobuf::internal::WireFormatLite;

template<
    int FieldNumber,
    WireFormatLite::FieldType FieldType,
    typename T,
    size_t (*SizeFunc)(T),
    void (*WriteNoTagFunc)(T, google::protobuf::io::CodedOutputStream*)
>
struct TValueTraitsImpl {
    using type = T;

    inline static size_t TagSize() {
        return WireFormatLite::TagSize(FieldNumber, FieldType);
    }

    inline static size_t SizeNoTag(T value) {
        return SizeFunc(value);
    }

    inline static void Write(T value, google::protobuf::io::CodedOutputStream* out) {
        WireFormatLite::WriteTag(FieldNumber, WireFormatLite::WireTypeForFieldType(FieldType), out);
        WriteNoTagFunc(value, out);
    }

    inline static void WriteNoTag(T value, google::protobuf::io::CodedOutputStream* out) {
        WriteNoTagFunc(value, out);
    }
};

template<int FieldNumber, WireFormatLite::FieldType FieldType>
struct TValueTraits {};

template<int FieldNumber>
struct TValueTraits<FieldNumber, WireFormatLite::TYPE_UINT64>
    : TValueTraitsImpl<
        FieldNumber,
        WireFormatLite::TYPE_UINT64,
        ui64,
        WireFormatLite::UInt64Size,
        WireFormatLite::WriteUInt64NoTag
    >
{};

template<int FieldNumber>
struct TValueTraits<FieldNumber, WireFormatLite::TYPE_INT64>
    : TValueTraitsImpl<
        FieldNumber,
        WireFormatLite::TYPE_INT64,
        i64,
        WireFormatLite::Int64Size,
        WireFormatLite::WriteInt64NoTag
    >
{};

template<int FieldNumber, WireFormatLite::FieldType FieldType>
requires (FieldType == WireFormatLite::TYPE_BYTES || FieldType == WireFormatLite::TYPE_STRING)
struct TValueTraits<FieldNumber, FieldType> {
    using type = std::string_view;

    inline static size_t TagSize() {
        return WireFormatLite::TagSize(FieldNumber, FieldType);
    }

    inline static void Write(std::string_view value, google::protobuf::io::CodedOutputStream* out) {
        WireFormatLite::WriteTag(FieldNumber, WireFormatLite::WireTypeForFieldType(FieldType), out);
        out->WriteVarint64(value.size());
        out->WriteRaw(value.data(), value.size());
    }
};

template<int FieldNumber, WireFormatLite::FieldType FieldType>
struct TValue {
    using TTraits = TValueTraits<FieldNumber, FieldType>;
    TTraits::type Value;

    inline static size_t TagSize() noexcept {
        return TTraits::TagSize();
    }

    inline size_t SizeNoTag() const noexcept {
        return TTraits::SizeNoTag(Value);
    }

    inline void Write(google::protobuf::io::CodedOutputStream* out) const noexcept {
        TTraits::Write(Value, out);
    }
};

template<typename... Values>
inline static size_t ValuesSize(const std::tuple<Values...>& values) {
    // I know, I know, there is std::apply. So what?
    return (Values::TagSize() + ...) + (std::get<Values>(values).SizeNoTag() + ...);
}

template<typename... Values>
inline static void ValuesWrite(const std::tuple<Values...>& values, google::protobuf::io::CodedOutputStream* out) {
    (std::get<Values>(values).Write(out), ...);
}

class TToPProfBytesConverterContext {
private:
    // Our new profile represntation is lossy.
    // We do not know exact addresess of mappings.
    static constexpr ui64 fakeMappingSize = 128_GB;

public:
    TToPProfBytesConverterContext(const NProto::NProfile::Profile& from, google::protobuf::io::CodedOutputStream* to)
        : Profile_(&from)
        , Out_(to)
    {}

    void WriteStrings() {
        using NProto::NPProf::Profile;

        for (auto&& str : Profile_.Strings()) {
            TValueTraits<Profile::kStringTableFieldNumber, WireFormatLite::TYPE_STRING>::Write(str.View(), Out_);
        }
    }

    void WriteBinaries() {
        using NProto::NPProf::Profile;
        using NProto::NPProf::Mapping;

        for (auto&& [i, binary] : Enumerate(Profile_.Binaries())) {
            if (i == 0) {
                // First binary is empty ant should not be present in pprof.
                continue;
            }

            std::tuple fields{
                TValue<Mapping::kIdFieldNumber, WireFormatLite::TYPE_UINT64>{i},
                TValue<Mapping::kBuildIdFieldNumber, WireFormatLite::TYPE_INT64>{*binary.GetBuildId().GetIndex()},
                TValue<Mapping::kFilenameFieldNumber, WireFormatLite::TYPE_INT64>{*binary.GetPath().GetIndex()},
                TValue<Mapping::kMemoryStartFieldNumber, WireFormatLite::TYPE_UINT64>{i * fakeMappingSize},
                TValue<Mapping::kMemoryLimitFieldNumber, WireFormatLite::TYPE_UINT64>{(i + 1) * fakeMappingSize},
            };

            WireFormatLite::WriteTag(Profile::kMappingFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
            Out_->WriteVarint64(ValuesSize(fields));
            ValuesWrite(fields, Out_);
        }
    }

    void WriteFunctions() {
        using NProto::NPProf::Profile;
        using NProto::NPProf::Function;

        for (auto&& [i, func] : Enumerate(Profile_.Functions())) {
            // Skip first function which must be empty.
            if (i == 0) {
                continue;
            }

            std::tuple fields{
                TValue<Function::kIdFieldNumber, WireFormatLite::TYPE_UINT64>{i},
                TValue<Function::kNameFieldNumber, WireFormatLite::TYPE_INT64>{*func.GetName().GetIndex()},
                TValue<Function::kSystemNameFieldNumber, WireFormatLite::TYPE_INT64>{*func.GetSystemName().GetIndex()},
                TValue<Function::kFilenameFieldNumber, WireFormatLite::TYPE_INT64>{*func.GetFileName().GetIndex()},
                TValue<Function::kStartLineFieldNumber, WireFormatLite::TYPE_INT64>{func.GetStartLine()},
            };

            WireFormatLite::WriteTag(Profile::kFunctionFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
            Out_->WriteVarint64(ValuesSize(fields));
            ValuesWrite(fields, Out_);
        }
    }

    void WriteStackFrames() {
        using NProto::NPProf::Profile;
        using NProto::NPProf::Location;
        using NProto::NPProf::Line;

        for (auto&& [i, frame] : Enumerate(Profile_.StackFrames())) {
            auto lineFields = [](auto&& line) {
                return std::tuple{
                    TValue<Line::kFunctionIdFieldNumber, WireFormatLite::TYPE_UINT64>{(ui64)*line.GetFunction().GetIndex()},
                    TValue<Line::kLineFieldNumber, WireFormatLite::TYPE_INT64>{line.GetLine()},
                    TValue<Line::kColumnFieldNumber, WireFormatLite::TYPE_INT64>{line.GetColumn()},
                };
            };

            size_t linesSize = 0;
            auto inlineChain = frame.GetInlineChain();
            for (auto&& line : inlineChain.GetLines()) {
                linesSize += WireFormatLite::TagSize(Location::kLineFieldNumber, WireFormatLite::TYPE_MESSAGE);
                linesSize += WireFormatLite::LengthDelimitedSize(ValuesSize(lineFields(line)));
            }

            ui64 mappingId = *frame.GetBinary().GetIndex();
            std::tuple fields{
                TValue<Location::kIdFieldNumber, WireFormatLite::TYPE_UINT64>{i + 1},
                TValue<Location::kMappingIdFieldNumber, WireFormatLite::TYPE_UINT64>{mappingId},
                TValue<Location::kAddressFieldNumber, WireFormatLite::TYPE_UINT64>{mappingId ? frame.GetAddress() + mappingId * fakeMappingSize : frame.GetAddress()},
            };

            WireFormatLite::WriteTag(Profile::kLocationFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
            Out_->WriteVarint64(ValuesSize(fields) + linesSize);
            ValuesWrite(fields, Out_);

            for (auto&& line : inlineChain.GetLines()) {
                auto fields = lineFields(line);
                WireFormatLite::WriteTag(Location::kLineFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
                Out_->WriteVarint64(ValuesSize(fields));
                ValuesWrite(fields, Out_);
            }
        }
    }

    void WriteComments() {
        using NProto::NPProf::Profile;

        size_t size = 0;
        for (auto&& comment : Profile_.Comments()) {
            size += TValueTraits<Profile::kCommentFieldNumber, WireFormatLite::TYPE_INT64>::SizeNoTag(*comment.GetString().GetIndex());
        }

        WireFormatLite::WriteTag(Profile::kCommentFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
        Out_->WriteVarint64(size);

        for (auto&& comment : Profile_.Comments()) {
            TValueTraits<Profile::kCommentFieldNumber, WireFormatLite::TYPE_INT64>::WriteNoTag(*comment.GetString().GetIndex(), Out_);
        }
    }

    void WriteMetadata() {
        using NProto::NPProf::Profile;

        const auto& metadata = Profile_.GetMetadata();

        // Write default_sample_type
        if (metadata.default_sample_type() > 0) {
            TValueTraits<Profile::kDefaultSampleTypeFieldNumber, WireFormatLite::TYPE_INT64>::Write(
                metadata.default_sample_type(), Out_);
        }

        // Write duration_nanos
        if (metadata.duration_nanoseconds() > 0) {
            TValueTraits<Profile::kDurationNanosFieldNumber, WireFormatLite::TYPE_INT64>::Write(
                metadata.duration_nanoseconds(), Out_);
        }

        // Write time_nanos
        if (metadata.has_timestamp()) {
            const auto& ts = metadata.timestamp();
            i64 timeNanos = ts.seconds() * 1'000'000'000 + ts.nanos();
            if (timeNanos > 0) {
                TValueTraits<Profile::kTimeNanosFieldNumber, WireFormatLite::TYPE_INT64>::Write(timeNanos, Out_);
            }
        }
    }

    void WriteSampleTypes() {
        using NProto::NPProf::Profile;
        using NProto::NPProf::ValueType;

        for (auto&& valueType : Profile_.ValueTypes()) {
            std::tuple fields{
                TValue<ValueType::kTypeFieldNumber, WireFormatLite::TYPE_INT64>{*valueType.GetType().GetIndex()},
                TValue<ValueType::kUnitFieldNumber, WireFormatLite::TYPE_INT64>{*valueType.GetUnit().GetIndex()},
            };

            WireFormatLite::WriteTag(Profile::kSampleTypeFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
            Out_->WriteVarint64(ValuesSize(fields));
            ValuesWrite(fields, Out_);
        }
    }

    void WriteSamples() {
        using NProto::NPProf::Profile;
        using NProto::NPProf::Sample;
        using NProto::NPProf::Label;

        for (auto&& sample : Profile_.Samples()) {
            // For now we write field even it is zero and singular.
            // This increases the size of the serialized profile slightly, but saves us some comparisons and additions in return.
            auto strLabelFields = [](i64 key, i64 str) {
                return std::tuple{
                    TValue<Label::kKeyFieldNumber, WireFormatLite::TYPE_INT64>{key},
                    TValue<Label::kStrFieldNumber, WireFormatLite::TYPE_INT64>{str},
                };
            };

            auto numLabelFields = [](i64 key, i64 num) {
                return std::tuple{
                    TValue<Label::kKeyFieldNumber, WireFormatLite::TYPE_INT64>{key},
                    TValue<Label::kNumFieldNumber, WireFormatLite::TYPE_INT64>{num},
                };
            };

            auto key = sample.GetKey();
            auto visitLabels = [&](auto&& visitor) {
                for (auto&& label : key.GetLabels()) {
                    if (label.IsString()) {
                        visitor(strLabelFields(*label.GetKey().GetIndex(), *label.GetString().GetIndex()));
                    } else {
                        visitor(numLabelFields(*label.GetKey().GetIndex(), label.GetNumber()));
                    }
                }
            };

            size_t locationIdsSize = 0;
            for (auto&& stack : key.GetStacks()) {
                for (auto&& frame : stack.GetFrames()) {
                    locationIdsSize += TValueTraits<Sample::kLocationIdFieldNumber, WireFormatLite::TYPE_UINT64>::SizeNoTag(*frame.GetIndex() + 1);
                }
            }

            size_t valuesSize = 0;
            for (auto&& sampleValue : sample.GetValues()) {
                valuesSize += TValueTraits<Sample::kValueFieldNumber, WireFormatLite::TYPE_INT64>::SizeNoTag(sampleValue.GetValue());
            }

            size_t size = WireFormatLite::TagSize(Sample::kLocationIdFieldNumber, WireFormatLite::TYPE_UINT64) + WireFormatLite::LengthDelimitedSize(locationIdsSize) +
                WireFormatLite::TagSize(Sample::kValueFieldNumber, WireFormatLite::TYPE_INT64) + WireFormatLite::LengthDelimitedSize(valuesSize);

            visitLabels([&](auto&& fields) {
                size += WireFormatLite::TagSize(Sample::kLabelFieldNumber, WireFormatLite::TYPE_MESSAGE) + WireFormatLite::LengthDelimitedSize(ValuesSize(fields));
            });

            WireFormatLite::WriteTag(Profile::kSampleFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
            Out_->WriteVarint64(size);

            WireFormatLite::WriteTag(Sample::kLocationIdFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
            Out_->WriteVarint64(locationIdsSize);

            for (auto&& stack : key.GetStacks()) {
                for (auto&& frame : stack.GetFrames()) {
                    TValueTraits<Sample::kLocationIdFieldNumber, WireFormatLite::TYPE_UINT64>::WriteNoTag(*frame.GetIndex() + 1, Out_);
                }
            }

            WireFormatLite::WriteTag(Sample::kValueFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
            Out_->WriteVarint64(valuesSize);

            for (auto&& sampleValue : sample.GetValues()) {
                TValueTraits<Sample::kValueFieldNumber, WireFormatLite::TYPE_INT64>::WriteNoTag(sampleValue.GetValue(), Out_);
            }

            visitLabels([&](auto&& fields) {
                WireFormatLite::WriteTag(Sample::kLabelFieldNumber, WireFormatLite::WIRETYPE_LENGTH_DELIMITED, Out_);
                Out_->WriteVarint64(ValuesSize(fields));
                ValuesWrite(fields, Out_);
            });
        }
    }

    void Convert() {
        WriteStrings();
        WriteBinaries();
        WriteFunctions();
        WriteStackFrames();
        WriteComments();
        WriteMetadata();
        WriteSampleTypes();
        WriteSamples();
    }

private:
    TProfile Profile_;
    google::protobuf::io::CodedOutputStream* Out_;
};

class TFromPProfConverterContext {
    enum class ESpecialMappingKind {
        None,
        Missing,
        Kernel,
        Python,
    };

public:
    explicit TFromPProfConverterContext(const NProto::NPProf::Profile& from, NProto::NProfile::Profile* to)
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
        ConvertMetadata();
        std::move(Builder_).Finish();
    }

    void ConvertMetadata() {
        // Convert default_sample_type if set
        if (OldProfile_.default_sample_type() > 0) {
            TStringId defaultType = ConvertString(OldProfile_.default_sample_type());
            Builder_.Metadata().GetProto().set_default_sample_type(*defaultType);
        }

        // Convert duration_nanos if set
        if (OldProfile_.duration_nanos() > 0) {
            Builder_.Metadata().GetProto().set_duration_nanoseconds(OldProfile_.duration_nanos());
        }

        // Convert time_nanos if set
        if (OldProfile_.time_nanos() > 0) {
            auto* timestamp = Builder_.Metadata().GetProto().mutable_timestamp();
            timestamp->set_seconds(OldProfile_.time_nanos() / 1'000'000'000);
            timestamp->set_nanos(OldProfile_.time_nanos() % 1'000'000'000);
        }
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
            BinaryMapping_.Add(mapping.id(), IntegerCast<ui32>(i), builder.Finish());

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
            FunctionMapping_.Add(function.id(), IntegerCast<ui32>(i), builder.Finish());
        }
    }

    void ConvertLocations() {
        Y_ABORT_UNLESS(LocationMapping_.IsEmpty());

        for (auto&& [i, location] : Enumerate(OldProfile_.location())) {
            Y_ENSURE(location.id() != 0, "Location id should be nonzero");

            ESpecialMappingKind kind = ClassifySpecialMapping(location);
            switch (kind) {
            case ESpecialMappingKind::None:
                break;

            case ESpecialMappingKind::Missing:
                OldMissingLocationIds_.Insert(location.id());
                break;

            case ESpecialMappingKind::Kernel:
                OldKernelLocationIds_.Insert(location.id());
                break;

            case ESpecialMappingKind::Python:
                OldPythonLocationIds_.Insert(location.id());
                break;
            }

            auto frame = Builder_.AddStackFrame();
            if (location.mapping_id()) {
                auto [mappingId, binaryId] = BinaryMapping_.GetPosition(location.mapping_id());
                auto&& mapping = OldProfile_.mapping(mappingId);
                ui64 address = location.address() + mapping.file_offset() - mapping.memory_start();

                frame.SetBinary(binaryId);
                frame.SetAddress(address);
            } else {
                frame.SetAddress(location.address());
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

            LocationMapping_.Add(location.id(), IntegerCast<ui32>(i), frame.Finish());
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
        auto keyBuilder = Builder_.AddSimpleSampleKey();
        ConvertSampleStack(keyBuilder, sample);
        ConvertSampleLabels(keyBuilder, sample);

        auto sampleBuilder = Builder_.AddSample();
        sampleBuilder.SetSampleKey(keyBuilder.Finish());
        ConvertSampleValues(sampleBuilder, sample);
        sampleBuilder.Finish();
    }

    void ConvertSampleStack(TProfileBuilder::TSimpleSampleKeyBuilder& builder, const NProto::NPProf::Sample& sample) {
        for (ui64 location : sample.location_id()) {
            TStackFrameId frame = LocationMapping_.GetNewIndex(location);
            builder.AddFrame(frame);
        }
    }

    ESpecialMappingKind ClassifySpecialMapping(const NProto::NPProf::Location& location) const {
        ui64 mappingId = location.mapping_id();

        if (mappingId == OldKernelMappingId_) {
            return ESpecialMappingKind::Kernel;
        } else if (mappingId == OldPythonMappingId_) {
            return ESpecialMappingKind::Python;
        } else if (mappingId == 0) {
            return ESpecialMappingKind::Missing;
        } else {
            return ESpecialMappingKind::None;
        }
    }

    void ConvertSampleValues(TProfileBuilder::TSampleBuilder& builder, const NProto::NPProf::Sample& sample) {
        for (auto&& [i, value] : Enumerate(sample.value())) {
            builder.AddValue(ValueTypes_.at(i), value);
        }
    }

    void ConvertSampleLabels(TProfileBuilder::TSimpleSampleKeyBuilder& keyBuilder, const NProto::NPProf::Sample& sample) {
        for (auto&& label : sample.label()) {
            TLabelId id = TLabelId::Invalid();
            if (0 != label.num()) {
                id = Builder_.AddNumericLabel(ConvertString(label.key()), label.num());
            } else {
                id = Builder_.AddStringLabel(ConvertString(label.key()), ConvertString(label.str()));
            }
            keyBuilder.AddLabel(id);
        }
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
    TCompactIntegerSet<ui64> OldMissingLocationIds_;
    TCompactIntegerSet<ui64> OldKernelLocationIds_;
    TCompactIntegerSet<ui64> OldPythonLocationIds_;
    ui64 OldKernelMappingId_ = Max<ui64>();
    ui64 OldPythonMappingId_ = Max<ui64>();
};

////////////////////////////////////////////////////////////////////////////////

class TToPProfConverterContext {
public:
    explicit TToPProfConverterContext(
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
        ConvertMetadata();
        ConvertMappings();
        ConvertFunctions();
        ConvertLocations();
        ConvertSamples();
    }

private:
    void ConvertStringTable() {
        for (auto str : SourceProfile_.Strings()) {
            TStringBuf view = str.View();

            OldProfile_.add_string_table(view);
        }

        Y_ENSURE(OldProfile_.string_table_size() > 0);
        Y_ENSURE(OldProfile_.string_table(0).empty());
    }

    int GetStringIndex(TStringBuf key) {
        int id = OldProfile_.string_table_size();
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

    void ConvertMetadata() {
        const auto& metadata = SourceProfile_.GetMetadata();

        // Convert default_sample_type
        if (metadata.default_sample_type() > 0) {
            OldProfile_.set_default_sample_type(metadata.default_sample_type());
        }

        // Convert duration_nanoseconds
        if (metadata.duration_nanoseconds() > 0) {
            OldProfile_.set_duration_nanos(metadata.duration_nanoseconds());
        }

        // Convert timestamp
        if (metadata.has_timestamp()) {
            const auto& ts = metadata.timestamp();
            i64 timeNanos = ts.seconds() * 1'000'000'000 + ts.nanos();
            if (timeNanos > 0) {
                OldProfile_.set_time_nanos(timeNanos);
            }
        }
    }

    void ConvertMappings() {
        auto binaries = SourceProfile_.Binaries();

        OldProfile_.mutable_mapping()->Reserve(binaries.size());
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
        auto functions = SourceProfile_.Functions();

        OldProfile_.mutable_function()->Reserve(functions.size());
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
        auto frames = SourceProfile_.StackFrames();

        // We add first null location as the "unknown" location and shift location ids by one.
        // pprof expects that Profile.sample.location_id are non-zero.
        OldProfile_.mutable_location()->Reserve(frames.size());
        for (auto [i, frame] : Enumerate(frames)) {
            NProto::NPProf::Location* location = OldProfile_.add_location();
            location->set_id(i + 1);

            auto inlineChain = frame.GetInlineChain();
            for (auto&& sourceLine : inlineChain.GetLines()) {
                NProto::NPProf::Line* line = location->add_line();
                line->set_function_id(*sourceLine.GetFunction().GetIndex());
                line->set_line(sourceLine.GetLine());
                line->set_column(sourceLine.GetColumn());
            }

            ui32 binaryId = *frame.GetBinary().GetIndex();
            ui64 address = frame.GetAddress();;
            if (binaryId == 0) {
                location->set_mapping_id(0);
                location->set_address(address);
            } else {
                const NProto::NPProf::Mapping& mapping = OldProfile_.mapping(binaryId - 1);

                // We need to build artificial address value.
                // See symmetric conversion in TConverterContext::ConvertLocations.
                ui64 adjustedAddress = address - mapping.file_offset() + mapping.memory_start();
                Y_ENSURE(adjustedAddress > 0);

                location->set_mapping_id(binaryId);
                location->set_address(adjustedAddress);
            }
        }
    }

    void ConvertSamples() {
        for (auto&& newSample : SourceProfile_.Samples()) {
            NProto::NPProf::Sample* oldSample = OldProfile_.add_sample();

            // Fill Sample.value
            for (auto&& sampleValue : newSample.GetValues()) {
                oldSample->add_value(sampleValue.GetValue());
            }

            // Fill Sample.stack
            ConvertSampleFrames(oldSample, newSample.GetKey());

            // Fill Sample.labels
            ConvertSampleLabels(oldSample, newSample.GetKey());
        }
    }

    void ConvertSampleFrames(NProto::NPProf::Sample* sample, const TSampleKey& key) {
        for (auto&& stack : key.GetStacks()) {
            for (auto&& frame : stack.GetFrames()) {
                // We shift location ids by 1 because pprof does not support zero location ids.
                // See corresponding comment inside ConvertLocations.
                sample->add_location_id(*frame.GetIndex() + 1);
            }
        }
    }

    void ConvertSampleLabels(NProto::NPProf::Sample* sample, TSampleKey key) {
        for (auto&& newLabel : key.GetLabels()) {
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

    void AddStringIdxLabel(NProto::NPProf::Sample* sample, TStringBuf key, TProfileString valueIdx) {
        AddLabel(sample, key)->set_str(*valueIdx.GetIndex());
    }

private:
    const NProfile::TProfile SourceProfile_;
    NProto::NPProf::Profile& OldProfile_;
};

} // namespace NDetail

void ConvertFromPProf(const NProto::NPProf::Profile& from, NProto::NProfile::Profile* to) {
    Y_ABORT_UNLESS(to, "Expected non-null output pointer");
    NDetail::TFromPProfConverterContext{from, to}.Convert();
}

namespace {

bool IsGzipCompressed(TStringBuf data) {
    // Gzip magic bytes: 0x1f 0x8b
    return data.size() >= 2 &&
        static_cast<ui8>(data[0]) == 0x1f &&
        static_cast<ui8>(data[1]) == 0x8b;
}

TString DecompressGzip(TStringBuf compressed) {
    TMemoryInput input{compressed.data(), compressed.size()};
    TZLibDecompress decompressor{&input};
    return decompressor.ReadAll();
}

} // namespace

void ConvertFromPProf(TStringBuf from, NProto::NProfile::Profile* to) {
    Y_ABORT_UNLESS(to, "Expected non-null output pointer");

    if (IsGzipCompressed(from)) {
        TString decompressed = DecompressGzip(from);
        NDetail::TFromPProfBytesConverterContext{decompressed, to}.Convert();
    } else {
        NDetail::TFromPProfBytesConverterContext{from, to}.Convert();
    }
}

void ConvertToPProf(const NProto::NProfile::Profile& from, NProto::NPProf::Profile* to) {
    Y_ABORT_UNLESS(to, "Expected non-null output pointer");
    NDetail::TToPProfConverterContext{from, to}.Convert();
}

void ConvertToPProf(const NProto::NProfile::Profile& from, TString* to) {
    Y_ABORT_UNLESS(to, "Expected non-null output pointer");
    google::protobuf::io::StringOutputStream strOut{to};
    google::protobuf::io::CodedOutputStream out{&strOut};
    NDetail::TToPProfBytesConverterContext{from, &out}.Convert();
}

} // namespace NPerforator
