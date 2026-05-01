#include "yc_refresh_token.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    TVector<TString> TYCRefreshToken::Patterns() const {
        return {
            R"(rt1\.[A-Z0-9a-z_\-]+[=]{0,2}\.[A-Z0-9a-z_\-]{86}[=]{0,2})",
        };
    }

    bool TYCRefreshToken::IsSecret(size_t id, const TStringBuf token) const {
        Y_UNUSED(id);

        if (NStringUtils::IsMasked(token, 80)) {
            return false;
        }

        const auto& parts = SplitString(TString(token), ".");
        if (parts.size() != 3) {
            return false;
        }

        return NStringUtils::IsBase64UrlRaw(parts[1]) && NStringUtils::IsBase64UrlRaw(parts[2]);
    }

    NSecret::TPos TYCRefreshToken::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        auto signature = rawSecret.RAfter('.').size();
        return {
            .From = rawSecret.length() - signature,
            .Len = signature,
        };
    }

    TString TYCRefreshToken::Name() const {
        return "yc-refresh-token";
    }

    NSecret::ESecretType TYCRefreshToken::SecretType() const {
        return NSecret::ESecretType::YCRefreshToken;
    }

    TMaybe<bool> TYCRefreshToken::ForceValid() const {
        return Nothing();
    }
}
