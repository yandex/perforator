#include "base62.h"
#include <util/generic/string.h>
#include <util/generic/ymath.h>
#include <library/cpp/digest/old_crc/crc.h>

namespace NBase62 {
    namespace {
        constexpr int kBase = 62.0;
        constexpr TStringBuf kDefaultCharset = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz";
    }

    template <typename T>
    T DecodeNum(TStringBuf in) {
        static_assert(std::is_integral_v<T>, "Only integer types are supported");
        T result = 0;
        size_t len = in.size();
        for (size_t i = 0; i < len; ++i) {
            size_t index = kDefaultCharset.find(in[i]);
            if (index == TString::npos) {
                return 0;
            }

            result += index * static_cast<size_t>(std::pow(kBase, len - (i+1)));
        }

        return result;
    }

    bool IsToken(TStringBuf in, size_t prefixLen, size_t crcLen) {
        if (in.size() <= prefixLen + crcLen) {
            return false;
        }

        ui32 expectedChecksum = DecodeNum<ui32>(in.Last(crcLen));
        if (expectedChecksum == 0) {
            return false;
        }

        auto entropy = in.Skip(prefixLen).Chop(crcLen);
        ui32 actualChecksum = crc32(entropy.data(), entropy.size());
        return actualChecksum == expectedChecksum;
    }
}
