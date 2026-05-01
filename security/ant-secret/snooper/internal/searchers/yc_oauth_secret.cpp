#include "yc_oauth_secret.h"

#include <security/ant-secret/internal/yc/yc.h>

namespace NSearchers {
    // https://a.yandex-team.ru/cloudia/cloud/cloud-java/iam/iam-control-plane/iam-oauth-client-secret-service/src/main/java/yandex/cloud/iam/cp/oauthclientsecret/OAuthClientSecretGenerator.java?rev=r334903#L10-31

    TVector<TString> TYCOAuthSecret::Patterns() const {
        return {
            R"(yccs__[0-9a-f]{64}_[0-9a-f]{8})",
        };
    }

    bool TYCOAuthSecret::IsSecret(size_t id, const TStringBuf token) const {
        Y_UNUSED(id);

        return NYC::ISOAuthSecret(token);
    }

    NSecret::TPos TYCOAuthSecret::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id, rawSecret);

        return {
            .From = rawSecret.length() - 73,
            .Len = 73,
        };
    }

    TString TYCOAuthSecret::Name() const {
        return "yc-oauth-secret";
    }

    NSecret::ESecretType TYCOAuthSecret::SecretType() const {
        return NSecret::ESecretType::YCOAuthSecret;
    }

    TMaybe<bool> TYCOAuthSecret::ForceValid() const {
        return false;
    }
}
