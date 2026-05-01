#include "tvm.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <library/cpp/string_utils/base64/base64.h>
#include <library/cpp/digest/old_crc/crc.h>

#include <util/string/cast.h>

namespace NTVM {
    namespace {
        constexpr size_t kMinSecretLen = 31;
        constexpr size_t kMinDecodedSecretLen = 20;
        constexpr ui8 kMaskLen = 27;
    }

    bool IsSecret(const TStringBuf token) {
        if (token.size() < kMinSecretLen) {
            return false;
        }

        bool isPrefixValid = (
            token[0] == 't' &&
            token[1] == 'v' &&
            token[2] == 'm' &&
            token[3] == '-'
        );
        if (!isPrefixValid) {
            return false;
        }

        if (NStringUtils::IsMasked(token, kMaskLen)) {
            return false;
        }

        TString decoded = Base64DecodeUneven(token);
        if (decoded.size() < kMinDecodedSecretLen) {
            return false;
        }

        ui32 expectedChecksum = NStringUtils::FromBytesMsb<ui32>(decoded, decoded.size() - 4);
        if (expectedChecksum == 0 || expectedChecksum > 4294967295) {
            return false;
        }

        TStringBuf rawSecret = TStringBuf(decoded).Skip(3);

        // It's important to use crc32, not crc32c
        ui32 actualChecksum = crc32(rawSecret.data(), rawSecret.size() - 4);
        return actualChecksum == expectedChecksum;
    }
}
