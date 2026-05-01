#include "jwt.h"

#include <security/ant-secret/internal/string_utils/common.h>
#include <security/ant-secret/internal/jwt/jwt.h>

#include <util/string/vector.h>


namespace NSearchers {
    TVector<TString> TJwt::Patterns() const {
        return {
            R"(ey[JKL][\w\-\\]{13,}\.ey[JKL][\w\-\\]{8,}\.[\w\-\\]{20,}[\w\-])",
        };
    }

    bool TJwt::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);
        return NJwt::IsToken(secret);
    }

    NSecret::TPos TJwt::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        auto signature = rawSecret.RAfter('.').size();
        return {
                .From = rawSecret.length() - signature,
                .Len = signature,
        };
    }

    TString TJwt::Name() const {
        return "jwt-token";
    }

    NSecret::ESecretType TJwt::SecretType() const {
        return NSecret::ESecretType::JwtToken;
    }

    TMaybe<bool> TJwt::ForceValid() const {
        return true;
    }
}
