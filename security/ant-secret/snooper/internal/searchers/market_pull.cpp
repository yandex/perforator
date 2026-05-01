#include "market_pull.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/split.h>
#include <util/string/hex.h>
#include <library/cpp/digest/old_crc/crc.h>

namespace NSearchers {
    namespace {
        constexpr size_t kPrefixLen = 5;
        constexpr size_t kCheckSumLen = 8;
    }

    TVector<TString> TMatketPull::Patterns() const {
        return {
            // https://st.yandex-team.ru/ANTSECRET-137
            R"(ACMA:[A-Za-z0-9_\-]{40}:[a-f0-9]{8})",
        };
    }

    bool TMatketPull::IsSecret(size_t id, const TStringBuf token) const {
        Y_UNUSED(id);

        if (token.size() < 54) {
            return false;
        }

        TString rawCRC32 = HexDecode(token.SubString(token.size() - kCheckSumLen, kCheckSumLen));
        if (rawCRC32.size() != 4) {
            return false;
        }

        ui32 expectedChecksum = NStringUtils::FromBytesMsb<ui32>(rawCRC32, 0);
        ui32 actualChecksum = crc32(token.data(), token.size() - kCheckSumLen);
        return actualChecksum == expectedChecksum;
    }

    NSecret::TPos TMatketPull::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id, rawSecret);

        return {
            .From = kPrefixLen,
            .Len = rawSecret.size() - kPrefixLen,
        };
    }

    TString TMatketPull::Name() const {
        return "market-pull";
    }

    NSecret::ESecretType TMatketPull::SecretType() const {
        return NSecret::ESecretType::MarketPull;
    }

    TMaybe<bool> TMatketPull::ForceValid() const {
        return Nothing();
    }
}
