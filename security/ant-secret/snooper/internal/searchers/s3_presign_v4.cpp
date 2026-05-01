#include "s3_presign_v4.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    TVector<TString> TS3PresignV4::KeyPatterns() const {
        // value with uri contain some like:
        // args=response-content-disposition=inline&response-content-type=application%2Fpdf&X-Amz-Content-Sha256=UNSIGNED-PAYLOAD&
        // X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=ffopfMqlKkU1fXiJHexn%2F20210721%2Fru-central1%2Fs3%2Faws4_request&
        // X-Amz-Date=20210721T084011Z&X-Amz-SignedHeaders=host&X-Amz-Expires=600&X-Amz-Signature=b98dfcaf857fa912636d8e7af81b426eb70299de2f69aabb0a38561642d204cb
        return {
            "x-amz-signature",
        };
    }

    TVector<TString> TS3PresignV4::ValuePatterns() const {
        return {
            R"([0-9a-z]{64})",
        };
    }

    bool TS3PresignV4::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);
        return secret.size() == 64;
    }

    NSecret::TPos TS3PresignV4::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        return {
            .From = rawSecret.length() - 64,
            .Len = 64,
        };
    }

    TString TS3PresignV4::Name() const {
        return "s3-presign";
    }

    NSecret::ESecretType TS3PresignV4::SecretType() const {
        return NSecret::ESecretType::S3Presign;
    }

    TMaybe<bool> TS3PresignV4::ForceValid() const {
        return true;
    }
}

