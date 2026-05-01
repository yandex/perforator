#include "ms_cookie.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    TVector<TString> TMsCookie::KeyPatterns() const {
        // https://st.yandex-team.ru/ANTSECRET-149
        return {
            "MSISAuth|MSISAuthenticated|FedAuth|EdgeAccessCookie",
        };
    }

    TVector<TString> TMsCookie::ValuePatterns() const {
        return {
            R"([A-Za-z0-9+\/=]{20,})",
        };
    }

    bool TMsCookie::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        if (NStringUtils::IsMasked(secret, secret.size() - 20)) {
            return false;
        }

        return NStringUtils::IsBase64Even(secret);
    }

    NSecret::TPos TMsCookie::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        return {
            .From = 0,
            .Len = rawSecret.length(),
        };
    }

    TString TMsCookie::Name() const {
        return "ms-cookie";
    }

    NSecret::ESecretType TMsCookie::SecretType() const {
        return NSecret::ESecretType::MsCookie;
    }

    TMaybe<bool> TMsCookie::ForceValid() const {
        return true;
    }
}

