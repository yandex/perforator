#include "yc_static_cred.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <library/cpp/string_utils/base64/base64.h>
#include <library/cpp/digest/crc32c/crc32c.h>

namespace NSearchers {
    TVector<TString> TYCStaticCred::Patterns() const {
        return {
            // https://st.yandex-team.ru/ANTSECRET-82
            R"(YC[A-Za-z0-9_\-]{38})",
        };
    }

    bool TYCStaticCred::IsSecret(size_t id, const TStringBuf token) const {
        Y_UNUSED(id);

        if (token.size() != 40) {
            return false;
        }

        TString decoded = Base64Decode(token);
        if (decoded.size() < 4) {
            return false;
        }

        ui32 expectedChecksum = NStringUtils::FromBytesMsb<ui32>(decoded, decoded.size() - 4);
        ui32 actualChecksum = Crc32c(decoded.data(), decoded.size() - 4);
        return actualChecksum == expectedChecksum;
    }

    NSecret::TPos TYCStaticCred::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id, rawSecret);

        return {
            .From = rawSecret.length() - 38,
            .Len = 38,
        };
    }

    TString TYCStaticCred::Name() const {
        return "yc-static-cred";
    }

    NSecret::ESecretType TYCStaticCred::SecretType() const {
        return NSecret::ESecretType::YCStaticCred;
    }

    TMaybe<bool> TYCStaticCred::ForceValid() const {
        return Nothing();
    }
}
