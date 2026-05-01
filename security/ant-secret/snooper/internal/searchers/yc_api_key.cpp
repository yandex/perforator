#include "yc_api_key.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <library/cpp/string_utils/base64/base64.h>

namespace NSearchers {
    TVector<TString> TYCApiKey::Patterns() const {
        return {
            // https://st.yandex-team.ru/ANTSECRET-82
            R"(AQVN[A-Za-z0-9_\-]{36})",
        };
    }

    bool TYCApiKey::IsSecret(size_t id, const TStringBuf token) const {
        Y_UNUSED(id);

        if (token.size() != 40) {
            return false;
        }

        if (NStringUtils::IsMasked(token, 36)) {
            return false;
        }

        TVector<unsigned char> decoded(Base64DecodeBufSize(token.size()));
        decoded.resize(Base64Decode(decoded.data(), token.begin(), token.end()));

        return (
                decoded.size() == 30 &&
                decoded[0] == 1 &&
                decoded[1] == 0x05 &&
                decoded[2] == 0x4d &&
                (decoded[3] >= 0xc0 && decoded[3] <= 0xdf)
        );
    }

    NSecret::TPos TYCApiKey::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id, rawSecret);

        return {
            .From = rawSecret.length() - 36,
            .Len = 36,
        };
    }

    TString TYCApiKey::Name() const {
        return "yc-apikey";
    }

    NSecret::ESecretType TYCApiKey::SecretType() const {
        return NSecret::ESecretType::YCApiKey;
    }

    TMaybe<bool> TYCApiKey::ForceValid() const {
        return Nothing();
    }
}
