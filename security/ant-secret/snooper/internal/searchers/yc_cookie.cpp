#include "yc_cookie.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>


namespace NSearchers {
    TVector<TString> TYCCookie::Patterns() const {
        return {
            // https://st.yandex-team.ru/ANTSECRET-82
            R"(c1\.[A-Z0-9a-z_\-]+[=]{0,2}\.[A-Z0-9a-z_\-]{86}[=]{0,2})",
        };
    }

    bool TYCCookie::IsSecret(size_t id, const TStringBuf token) const {
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

    NSecret::TPos TYCCookie::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        auto signature = rawSecret.RAfter('.').size();
        return {
            .From = rawSecret.length() - signature,
            .Len = signature,
        };
    }

    TString TYCCookie::Name() const {
        return "yc-cookie";
    }

    NSecret::ESecretType TYCCookie::SecretType() const {
        return NSecret::ESecretType::YCCookie;
    }

    TMaybe<bool> TYCCookie::ForceValid() const {
        return Nothing();
    }
}
