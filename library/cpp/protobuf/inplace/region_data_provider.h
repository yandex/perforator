#pragma once

#include "serialized.h"

#include <util/system/byteorder.h>

namespace NInPlaceProto {

    class TRegionDataProvider {
    private:
        const ui8* Start = nullptr;
        const ui8*const End = nullptr;
        bool Corrupted = false;

    public:
        template <typename TProtoMessage>
        explicit TRegionDataProvider(TSerialized<TProtoMessage> serialized)
            : Start(serialized.Data())
            , End(Start + serialized.Size())
        {
        }
        TRegionDataProvider(const ui8* start, size_t len)
            : Start(start)
            , End(Start + len)
        {
        }
        TRegionDataProvider(const char* start, size_t len)
            : Start((const ui8*)start)
            , End(Start + len)
        {
        }
        TRegionDataProvider(const ui8* start, const ui8* end)
            : Start(start)
            , End(end)
        {
        }
        TRegionDataProvider(const char* start, const char* end)
            : Start((const ui8*)start)
            , End((const ui8*)end)
        {
        }

        bool IsCorrupted() const noexcept {
            return Corrupted;
        }
        void SetCorrupted() noexcept {
            Corrupted = true;
        }
        void UpdateCorrupted(bool corrupted) noexcept {
            Corrupted |= corrupted;
        }

        bool NotEmpty() const noexcept {
            return Start < End;
        }
        bool CanRead(size_t n) const noexcept {
            return Start + n <= End;
        }

        void SkipVarint64() noexcept {
            int count = 0;
            ui32 b;
            const ui8* start = Start;
            const ui8*const end = End;
            do {
                if (count == 10 || start >= end) {
                    Corrupted = true;
                    break;
                }
                b = *start++;
            } while (b >= 0x80);
            Start = start;
        }
        ui32 ReadVarint32Slow(ui32 firstByte) noexcept;
        inline ui32 ReadVarint32() noexcept {
            if (Start < End) {
                ui32 res = *Start++;
                if (res < 0x80) {
                    return res;
                }
                return ReadVarint32Slow(res);
            }
            Corrupted = true;
            return 0;
        }
        inline ui32 ReadTag() noexcept {
            const bool corrupted = Corrupted;
            const ui8*const start = Start;
            const ui8*const end = End;
            if (Y_LIKELY(!corrupted)) {
                if (Y_LIKELY(start < end)) {
                    ui32 res = *start;
                    ++Start;
                    if (res < 0x80) {
                        return res;
                    }
                    return ReadVarint32Slow(res);
                }
            }
            return 0;
        }

        inline bool ReadVarintBool() {
            if (Start < End) {
                ui32 res = *Start++;
                if (res < 0x80) {
                    return res;
                }
                for (int count = 0; count < 9; ++count) {
                    if (*Start++ < 0x80) {
                        return true;
                    }
                }
            }
            Corrupted = true;
            return false;
        }

        // Do not support huge tag ids
        template <ui16 tag>
        bool PeekTag14() noexcept {
            static_assert(tag < 128 * 128, "Huge tag ids can't be easily looked up");
            const bool corrupted = Corrupted;
            const ui8*const start = Start;
            const ui8*const end = End;
            if (Y_LIKELY(!corrupted)) {
                if constexpr (tag < 128) {
                    if (Y_LIKELY(start < end)) {
                        if (Y_LIKELY(*start == tag)) {
                            Start = start + 1;
                            return true;
                        }
                    }
                } else {
                    if (Y_LIKELY(start + 2 <= end)) {
                        ui16 firstByte = tag & 0x7F | 0x80;
                        ui16 secondByte = tag >> 7 & 0x7F;
                        ui16 encodedTag = secondByte << 8 | firstByte;
                        if (Y_LIKELY(*(const ui16*)start == HostToLittle(encodedTag))) {
                            Start = start + 2;
                            return true;
                        }
                    }
                }
            }
            return false;
        }

        bool PeekTag8(ui8 tag) noexcept {
            const bool corrupted = Corrupted;
            const ui8*const start = Start;
            const ui8*const end = End;
            if (Y_LIKELY(!corrupted)) {
                if (Y_LIKELY(start < end)) {
                    if (Y_LIKELY(*start == tag)) {
                        Start = start + 1;
                        return true;
                    }
                }
            }
            return false;
        }
        ui8 RememberTag8Unsafe() noexcept {
            return Start[-1];
        }

        // Expect ui16 to be in fact two sequential bytes of tag
        // So, no they are in little endian regardless of host endiannes
        bool PeekTag16(ui16 tag) noexcept {
            const bool corrupted = Corrupted;
            const ui8*const start = Start;
            const ui8*const end = End;
            if (Y_LIKELY(!corrupted)) {
                if (Y_LIKELY(start + 2 <= end)) {
                    if (Y_LIKELY(*(const ui16*)start == tag)) {
                        Start = start + 2;
                        return true;
                    }
                }
            }
            return false;
        }
        ui16 RememberTag16Unsafe() noexcept {
            return *(ui16*)(Start - 2);
        }

        ui64 ReadVarint64Slow() noexcept;
        ui64 ReadVarint64() noexcept {
            if (Y_LIKELY(Start < End)) {
                ui32 res = *Start;
                if (Y_LIKELY(res < 0x80)) {
                    ++Start;
                    return res;
                }
                return ReadVarint64Slow();
            }
            Corrupted = true;
            return 0;
        }
        ui32 ReadLittleEndian32() noexcept {
            if (Y_LIKELY(Start + sizeof(ui32) <= End)) {
                ui32 value = LittleToHost(*(const ui32*)Start);
                Start += sizeof(ui32);
                return value;
            }
            Corrupted = true;
            return 0;
        }
        ui64 ReadLittleEndian64() noexcept {
            if (Y_LIKELY(Start + sizeof(ui64) <= End)) {
                ui64 value = LittleToHost(*(const ui64*)Start);
                Start += sizeof(ui64);
                return value;
            }
            Corrupted = true;
            return 0;
        }
        void Skip(ui32 length) noexcept {
            if (Y_LIKELY(Start + length <= End)) {
                Start += length;
            } else {
                Corrupted = true;
            }
        }

        TArrayRef<const char> GetRegion(ui32 length) noexcept {
            if (Y_LIKELY(Start + length <= End)) {
                const ui8*const oldStart = Start;
                Start += length;
                return TArrayRef<const char>((const char*)oldStart, (const char*)Start);
            }
            Corrupted = true;
            return TArrayRef<const char>();
        }
        // to track unknown fields (to pass them unchanged to somewhere else),
        // call GetCurrentPos() before ReadTag() and after Skip*()
        const ui8* GetCurrentPos() const noexcept {
            return Start;
        }
    };

} // namespace NInPlaceProto
