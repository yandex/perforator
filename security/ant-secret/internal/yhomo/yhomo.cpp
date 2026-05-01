#include "yhomo.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <library/cpp/string_utils/base64/base64.h>
#include <library/cpp/digest/crc32c/crc32c.h>
#include <security/ant-secret/pwdgen/pkg/pwd/meta.pb.h>

#include <util/string/split.h>
#include <util/string/hex.h>

namespace NYHomo {
    constexpr size_t kMinPassLen = 16;

    bool IsPassword(const TStringBuf token) {
        if (token.size() < kMinPassLen) {
            return false;
        }

        auto parts = StringSplitter(token).Split(':').ToList<TStringBuf>();
        if (parts.size() != 4) {
            return false;
        }

        TString rawMeta = Base64DecodeUneven(parts[1]);
        if (rawMeta.size() < 8) {
            return false;
        }

        NPwdGen::NPwd::Meta meta;
        if (!meta.ParseFromString(rawMeta)) {
            return false;
        }

        if (meta.ts() <= 0 || meta.notify_id().empty()) {
            return false;
        }

        TString rawCRC32 = HexDecode(parts[3]);
        if (rawCRC32.size() != 4) {
            return false;
        }

        ui32 expectedChecksum = NStringUtils::FromBytesMsb<ui32>(rawCRC32, 0);
        if (expectedChecksum == 0 || expectedChecksum > 4294967295) {
            return false;
        }

        ui32 actualChecksum = Crc32c(token.data(), token.size() - 8);
        return actualChecksum == expectedChecksum;
    }
}
