#include "s3_secret_key.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <library/cpp/string_utils/base64/base64.h>
#include <library/cpp/digest/old_crc/crc.h>

namespace NSearchers {
    TVector<TString> TS3SecretKey::Patterns() const {
        return {
            // https://st.yandex-team.ru/MDS-29657
            R"(S3[A-Za-z0-9+/]{38})",
        };
    }

    bool TS3SecretKey::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        if (secret.size() != 40) {
            return false;
        }

        if (NStringUtils::IsMasked(secret, secret.size()/2)) {
            return false;
        }

        TString decoded = Base64Decode(secret);
        if (decoded.size() < 4) {
            return false;
        }

        ui32 expectedChecksum = NStringUtils::FromBytesMsb<ui32>(decoded, decoded.size() - 4);
        ui32 actualChecksum = crc32(decoded.data(), decoded.size() - 4);
        return actualChecksum == expectedChecksum;
    }

    NSecret::TPos TS3SecretKey::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id, rawSecret);

        return {
            .From = rawSecret.length() - 38,
            .Len = 38,
        };
    }

    TString TS3SecretKey::Name() const {
        return "s3-secret-key";
    }

    NSecret::ESecretType TS3SecretKey::SecretType() const {
        return NSecret::ESecretType::S3SecretKey;
    }

    TMaybe<bool> TS3SecretKey::ForceValid() const {
        return false;
    }
}
