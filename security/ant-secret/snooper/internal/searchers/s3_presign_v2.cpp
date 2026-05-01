#include "s3_presign_v2.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    TVector<TString> TS3PresignV2::KeyPatterns() const {
        // value with uri contain some like:
        // AWSAccessKeyId=Emzb9VBoUXGbVnOLHz0u&Signature=ldoFoSgyZaO3aRRC3dwitGH9p%2Fs%3D&Expires=1744292737
        return {
            "signature",
        };
    }

    TVector<TString> TS3PresignV2::ValuePatterns() const {
        return {
            R"((?:[A-Za-z0-9+/=]|%2[bB]|%2[fF]|%3[dD]){28})",
        };
    }

    bool TS3PresignV2::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        if (NStringUtils::IsMasked(secret, secret.size())) {
            return false;
        }

        return NStringUtils::IsBase64Even(secret);
    }

    NSecret::TPos TS3PresignV2::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        return {
            .From = 0,
            .Len = rawSecret.length(),
        };
    }

    TString TS3PresignV2::Name() const {
        return "s3-presign";
    }

    NSecret::ESecretType TS3PresignV2::SecretType() const {
        return NSecret::ESecretType::S3Presign;
    }

    TMaybe<bool> TS3PresignV2::ForceValid() const {
        return true;
    }
}
