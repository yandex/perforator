#include "dev_apikey.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    TVector<TString> TDevApiKey::KeyPatterns() const {
        // https://st.yandex-team.ru/ANTSECRET-154
        return {
            "apikey",
        };
    }

    TVector<TString> TDevApiKey::ValuePatterns() const {
        // uuid4
        return {
            R"([0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12})",
        };
    }

    bool TDevApiKey::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        return !NStringUtils::IsMasked(secret, secret.size());
    }

    NSecret::TPos TDevApiKey::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        // left left and right values
        return {
            .From = 9,
            .Len = rawSecret.length() - 9 - 13,
        };
    }

    TString TDevApiKey::Name() const {
        return "dev-apikey";
    }

    NSecret::ESecretType TDevApiKey::SecretType() const {
        return NSecret::ESecretType::DevApiKey;
    }

    TMaybe<bool> TDevApiKey::ForceValid() const {
        return true;
    }
}
