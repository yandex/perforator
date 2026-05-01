#pragma once

#include <util/generic/strbuf.h>
#include <util/generic/yexception.h>

namespace NStringUtils {
    bool IsHash(const TStringBuf data);

    bool IsBase64(const TStringBuf data);

    bool IsBase64Even(const TStringBuf data);

    bool IsBase64Url(const TStringBuf data);

    bool IsBase64UrlRaw(const TStringBuf data);

    bool IsBase64UrlEven(const TStringBuf data);

    bool IsMasked(const TStringBuf data, size_t maskLen);

    bool IsValidUTF8(const TStringBuf data);

    // Big Endian
    template <typename T>
    T FromBytesMsb(TStringBuf buf, size_t pos = 0) {
        static_assert(std::is_integral_v<T>, "Only integer types are supported");
        Y_ENSURE(pos + sizeof(T) <= buf.size(), "Buffer position out of range");
        T result = 0;
        for (size_t i = 0; i < sizeof(T); ++i) {
            ui8 byte = buf[pos + i];
            result |= ((T)byte) << (8 * (sizeof(T) - 1 - i));
        }
        return result;
    }
}
