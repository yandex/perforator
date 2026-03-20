#pragma once

#include "packed_repeated_view.h"
#include "serialized.h"

#include <util/generic/vector.h>

#include <google/protobuf/wire_format_lite.h>

namespace NInPlaceProto {

    // Usage pattern:
    // while (ui32 fieldNumber = parser.NextFieldNumber()) {
    //     switch(fieldNumber) {
    //         case PROTO_FIELD_ID(TMyMessage, MyField1): {
    //             someVar1 = parser.GetFixed64();
    //             break;
    //         }
    //         ..
    //         case PROTO_FIELD_ID(TMyMessage, MyFieldN): {
    //             someVarN = parser.GetStringAsBuf();
    //             break;
    //         }
    //         default: {
    //             parser.SkipField();
    //             break;
    //         }
    //     }
    // }

    template <typename TDataProvider>
    class TInplaceParser : public TDataProvider {
    private:
        using WireFormatLite = ::google::protobuf::internal::WireFormatLite;

        ui32 FieldNumber = 0;
        WireFormatLite::WireType WireType;

        ui32 GetVarint32Impl() {
            if (Y_LIKELY(WireType == WireFormatLite::WIRETYPE_VARINT)) {
                return TDataProvider::ReadVarint32();
            } else if (WireType == WireFormatLite::WIRETYPE_FIXED32) {
                return TDataProvider::ReadLittleEndian32();
            } else if (WireType == WireFormatLite::WIRETYPE_FIXED64) {
                return TDataProvider::ReadLittleEndian64();
            } else {
                TDataProvider::SetCorrupted();
                return 0;
            }
        }
        template <typename T32, typename T64>
        ui64 GetVarint64Impl() {
            if (Y_LIKELY(WireType == WireFormatLite::WIRETYPE_VARINT)) {
                return TDataProvider::ReadVarint64();
            } else if (WireType == WireFormatLite::WIRETYPE_FIXED32) {
                return (T64)(T32)TDataProvider::ReadLittleEndian32();
            } else if (WireType == WireFormatLite::WIRETYPE_FIXED64) {
                return TDataProvider::ReadLittleEndian64();
            } else {
                TDataProvider::SetCorrupted();
                return 0;
            }
        }
        ui32 GetFixed32Impl() {
            if (Y_LIKELY(WireType == WireFormatLite::WIRETYPE_FIXED32)) {
                return TDataProvider::ReadLittleEndian32();
            } else if (WireType == WireFormatLite::WIRETYPE_VARINT) {
                return TDataProvider::ReadVarint32();
            } else if (WireType == WireFormatLite::WIRETYPE_FIXED64) {
                return TDataProvider::ReadLittleEndian64();
            } else {
                TDataProvider::SetCorrupted();
                return 0;
            }
        }
        template <typename T32, typename T64>
        ui64 GetFixed64Impl() {
            if (Y_LIKELY(WireType == WireFormatLite::WIRETYPE_FIXED64)) {
                return TDataProvider::ReadLittleEndian64();
            } else if (WireType == WireFormatLite::WIRETYPE_VARINT) {
                return TDataProvider::ReadVarint64();
            } else if (WireType == WireFormatLite::WIRETYPE_FIXED32) {
                return (T64)(T32)TDataProvider::ReadLittleEndian32();
            } else {
                TDataProvider::SetCorrupted();
                return 0;
            }
        }

    public:
        using TDataProvider::TDataProvider;

        ui32 NextFieldNumber() noexcept {
            ui32 tag = TDataProvider::ReadTag();
            WireType = static_cast<WireFormatLite::WireType>(tag & 0x7); // kTagTypeMask
            return FieldNumber = tag >> 3; // kTagTypeBits
        }

        ui32 GetFieldNumber() noexcept {
            return FieldNumber;
        }

        template <ui32 tagId, WireFormatLite::WireType wireType>
        bool PeekTagId2048() noexcept {
            static_assert(tagId < 2048); // 2^7 * 2^7 / 8
            constexpr ui32 tag = WireFormatLite::MakeTag(tagId, wireType);
            if (TDataProvider::template PeekTag14<tag>()) {
                FieldNumber = tagId;
                WireType = wireType;
                return true;
            }
            return false;
        }

        double GetDouble() noexcept {
            if (Y_LIKELY(WireType == WireFormatLite::WIRETYPE_FIXED64)) {
                ui64 tmp = TDataProvider::ReadLittleEndian64();
                return WireFormatLite::DecodeDouble(tmp);
            }
            TDataProvider::SetCorrupted();
            return 0;
        }
        float GetFloat() noexcept {
            if (Y_LIKELY(WireType == WireFormatLite::WIRETYPE_FIXED32)) {
                ui32 tmp = TDataProvider::ReadLittleEndian32();
                return WireFormatLite::DecodeFloat(tmp);
            }
            TDataProvider::SetCorrupted();
            return 0;
        }
        i32 GetInt32() noexcept {
            return GetVarint32Impl();
        }
        i64 GetInt64() noexcept {
            return GetVarint64Impl<i32, i64>();
        }
        ui32 GetUInt32() noexcept {
            return GetVarint32Impl();
        }
        ui64 GetUInt64() noexcept {
            return GetVarint64Impl<ui32, ui64>();
        }
        i32 GetSInt32() noexcept {
            if (Y_LIKELY(WireType == WireFormatLite::WIRETYPE_VARINT)) {
                ui32 tmp = TDataProvider::ReadVarint32();
                return static_cast<i32>(tmp >> 1) ^ -static_cast<i32>(tmp & 1);
            }
            TDataProvider::SetCorrupted();
            return 0;
        }
        i64 GetSInt64() noexcept {
            if (Y_LIKELY(WireType == WireFormatLite::WIRETYPE_VARINT)) {
                ui64 tmp = TDataProvider::ReadVarint64();
                return static_cast<i64>(tmp >> 1) ^ -static_cast<i64>(tmp & 1);
            }
            TDataProvider::SetCorrupted();
            return 0;
        }
        ui32 GetFixed32() noexcept {
            return GetFixed32Impl();
        }
        ui64 GetFixed64() noexcept {
            return GetFixed64Impl<ui32, ui64>();
        }
        i32 GetSFixed32() noexcept {
            return (i32)GetFixed32Impl();
        }
        i64 GetSFixed64() noexcept {
            return GetFixed64Impl<i32, i64>();
        }
        bool GetBool() noexcept {
            return GetVarint64Impl<ui32, ui64>();
        }
        TStringBuf GetStringAsBuf() noexcept {
            if (WireType == WireFormatLite::WIRETYPE_LENGTH_DELIMITED) {
                ui32 length = TDataProvider::ReadVarint32();
                TArrayRef<const char> region = TDataProvider::GetRegion(length);
                return TStringBuf(region.data(), region.size());
            }
            TDataProvider::SetCorrupted();
            return TStringBuf();
        }
        TString GetString() {
            if (WireType == WireFormatLite::WIRETYPE_LENGTH_DELIMITED) {
                ui32 length = TDataProvider::ReadVarint32();
                TArrayRef<const char> region = TDataProvider::GetRegion(length);
                return TString(region.data(), region.size());
            }
            TDataProvider::SetCorrupted();
            return TString();
        }
        TStringBuf GetBytesAsBuf() noexcept {
            return GetStringAsBuf();
        }
        TString GetBytes() {
            return GetString();
        }

        template <typename TProtoMessage>
        TSerialized<TProtoMessage> GetSerialized() noexcept {
            return AsSerialized<TProtoMessage>(GetStringAsBuf());
        }

        template <EPackedType PackedType>
        TPackedRepeatedView<PackedType> GetPackedRepeatedView() noexcept {
            if (WireType == WireFormatLite::WIRETYPE_LENGTH_DELIMITED) {
                ui32 length = TDataProvider::ReadVarint32();
                TArrayRef<const char> region = TDataProvider::GetRegion(length);
                return TPackedRepeatedView<PackedType>(region, *this);
            }
            TDataProvider::SetCorrupted();
            return TPackedRepeatedView<PackedType>(TStringBuf{}, *this);
        }

        template <EPackedType PackedType>
        TVector<typename NPackedRepeatedDetail::TPackedTypeTraits<PackedType>::TValue> GetPackedRepeated() {
            using TTraits = NPackedRepeatedDetail::TPackedTypeTraits<PackedType>;
            using T = typename TTraits::TValue;
            auto view = GetPackedRepeatedView<PackedType>();
            TVector<T> result;
            if constexpr (TTraits::IsFixed) {
                result.reserve(view.size());
            } else {
                result.reserve(NPackedRepeatedDetail::CountVarintElements(
                    view.GetRawBuf().data(), view.GetRawBuf().size()));
            }
            for (auto x : view) {
                result.push_back(x);
            }
            return result;
        }

        // For advanced usage only. Generic parsers & etc
        WireFormatLite::WireType GetWireType() const noexcept {
            return WireType;
        }

        void SkipField() noexcept {
            switch (WireType) {
                case WireFormatLite::WIRETYPE_VARINT: {
                    TDataProvider::SkipVarint64();
                    break;
                }
                case WireFormatLite::WIRETYPE_FIXED64: {
                    TDataProvider::Skip(sizeof(ui64));
                    break;
                }
                case WireFormatLite::WIRETYPE_LENGTH_DELIMITED: {
                    ui32 length = TDataProvider::ReadVarint32();
                    TDataProvider::Skip(length);
                    break;
                }
                case WireFormatLite::WIRETYPE_START_GROUP: {
                    SkipGroup(FieldNumber);
                    break;
                }
                case WireFormatLite::WIRETYPE_END_GROUP: {
                    TDataProvider::SetCorrupted();
                    return;
                }
                case WireFormatLite::WIRETYPE_FIXED32: {
                    TDataProvider::Skip(sizeof(ui32));
                    break;
                }
                default:
                    TDataProvider::SetCorrupted();
                    return;
            }
        }

        bool TrySkipMulti() noexcept {
            WireFormatLite::WireType wireType = WireType;
            if (FieldNumber < 16) {
                // single byte hack
                ui8 tag = TDataProvider::RememberTag8Unsafe();
                switch (wireType) {
                    case WireFormatLite::WIRETYPE_VARINT: {
                        do {
                            TDataProvider::SkipVarint64();
                        } while (TDataProvider::PeekTag8(tag));
                        break;
                    }
                    case WireFormatLite::WIRETYPE_FIXED64: {
                        do {
                            TDataProvider::Skip(sizeof(ui64));
                        } while (TDataProvider::PeekTag8(tag));
                        break;
                    }
                    case WireFormatLite::WIRETYPE_LENGTH_DELIMITED: {
                        do {
                            ui32 length = TDataProvider::ReadVarint32();
                            TDataProvider::Skip(length);
                        } while (TDataProvider::PeekTag8(tag));
                        break;
                    }
                    case WireFormatLite::WIRETYPE_FIXED32: {
                        do {
                            TDataProvider::Skip(sizeof(ui32));
                        } while (TDataProvider::PeekTag8(tag));
                        break;
                    }
                    default:
                        return false;
                }
                return true;
            } else if (FieldNumber < 16 * 128) {
                // two bytes hack
                ui16 tag = TDataProvider::RememberTag16Unsafe();
                switch (wireType) {
                    case WireFormatLite::WIRETYPE_VARINT: {
                        do {
                            TDataProvider::SkipVarint64();
                        } while (TDataProvider::PeekTag16(tag));
                        break;
                    }
                    case WireFormatLite::WIRETYPE_FIXED64: {
                        do {
                            TDataProvider::Skip(sizeof(ui64));
                        } while (TDataProvider::PeekTag16(tag));
                        break;
                    }
                    case WireFormatLite::WIRETYPE_LENGTH_DELIMITED: {
                        do {
                            ui32 length = TDataProvider::ReadVarint32();
                            TDataProvider::Skip(length);
                        } while (TDataProvider::PeekTag16(tag));
                        break;
                    }
                    case WireFormatLite::WIRETYPE_FIXED32: {
                        do {
                            TDataProvider::Skip(sizeof(ui32));
                        } while (TDataProvider::PeekTag16(tag));
                        break;
                    }
                    default:
                        return false;
                }
                return true;
            }
            return false;
        }

        void SkipMulti() noexcept {
            if (!TrySkipMulti()) {
                SkipField();
            }
        }

        // For advanced usage only. Generic parsers & etc
        void SkipGroup(ui32 endFieldNumber) noexcept {
            while (TDataProvider::NotEmpty()) {
                ui32 tag = TDataProvider::ReadTag();
                const ui32 fieldNumber = WireFormatLite::GetTagFieldNumber(tag);
                const WireFormatLite::WireType wireType = WireFormatLite::GetTagWireType(tag);
                if (fieldNumber == 0) {
                    return;
                }
                switch (wireType) {
                    case WireFormatLite::WIRETYPE_VARINT: {
                        TDataProvider::SkipVarint64();
                        break;
                    }
                    case WireFormatLite::WIRETYPE_FIXED64: {
                        TDataProvider::Skip(sizeof(ui64));
                        break;
                    }
                    case WireFormatLite::WIRETYPE_LENGTH_DELIMITED: {
                        ui32 length = TDataProvider::ReadVarint32();
                        TDataProvider::Skip(length);
                        break;
                    }
                    case WireFormatLite::WIRETYPE_START_GROUP: {
                        SkipGroup(fieldNumber);
                        break;
                    }
                    case WireFormatLite::WIRETYPE_END_GROUP: {
                        if (fieldNumber != endFieldNumber) {
                            TDataProvider::SetCorrupted();
                        }
                        return;
                    }
                    case WireFormatLite::WIRETYPE_FIXED32: {
                        TDataProvider::Skip(sizeof(ui32));
                        break;
                    }
                    default:
                        TDataProvider::SetCorrupted();
                        return;
                }
            }
        }
    };

} // namespace NInPlaceProto
