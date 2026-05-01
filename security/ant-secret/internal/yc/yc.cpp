#include "yc.h"

#include <security/ant-secret/internal/digest/murmur3.h>
#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/hex.h>

namespace NYC {
    constexpr size_t kMinLen = 79;

    bool ISOAuthSecret(const TStringBuf secret) {
        if (secret.size() != kMinLen) {
            return false;
        }

        size_t pos = secret.rfind('_');
        if (pos == std::string::npos) {
            return false;
        }

        TStringBuf encodedHash = secret.substr(pos + 1);
        TString rawHash = HexDecode(encodedHash);
        if (rawHash.size() != 4) {
            return false;
        }

        ui32 expectedHash = NStringUtils::FromBytesMsb<ui32>(rawHash, 0);
        if (expectedHash == 0) {
            return false;
        }

        TStringBuf body = secret.substr(0, pos + 1);
        uint32_t actualHash = NDigest::Murmur3_32(body, 0);
        return actualHash == expectedHash;
    }
}
