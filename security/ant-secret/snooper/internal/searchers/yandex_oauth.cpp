#include "yandex_oauth.h"

#include <security/ant-secret/internal/string_utils/common.h>
#include <security/ant-secret/internal/blackbox/oauth.h>


namespace NSearchers {
    namespace {
        enum EPattern {
            kEmbeddedV2ID,
            kEmbeddedV3ID,
            kStatelessV1ID,
            kStatelessV2ID,
        };
    }

    TVector<TString> TYandexOAuth::Patterns() const {
        return {
            // embedded v2
            R"(A[\w\-]{37}[A-Za-z0-9])",
            // embedded v3+
            R"(y[0-9]\\?_([A-Za-z0-9\-]|\\?_){37,199}[\w\-])",
            // stateless v1
            R"(1\.\d+\.\d+\.\d+\.\d+\.\d+\.[\w\-]{16}\.[\w\-]+\.[\w\-]{22})",
            // stateless v2
            R"(2\.\d+\.\d+\.\d+\.\d+\.\d+\.\d+\.\d+\.[\w\-]{16}\.[\w\-]+\.[\w\-]{22})",
        };
    }

    bool TYandexOAuth::IsSecret(size_t id, const TStringBuf token) const {
        switch (id) {
            case kEmbeddedV2ID:
                return NBlackBox::IsEmbeddedV2Token(token);
            case kEmbeddedV3ID:
                return NBlackBox::IsEmbeddedV3Token(token);
            case kStatelessV1ID:
                [[fallthrough]];
            case kStatelessV2ID:
                return token.size() > 70 && !NStringUtils::IsMasked(token, 22);
            default:
                ythrow TSystemError() << "unexpected token pattern id: " << id;
        }
    }

    NSecret::TPos TYandexOAuth::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        switch (id) {
            case kEmbeddedV2ID:
                // we left shard + uid + clientid
                return {
                    .From = rawSecret.length() - 21,
                    .Len = 21,
                };
            case kEmbeddedV3ID:
                // we mask only random + crc32 part
                return {
                    .From = rawSecret.length() - 27,
                    .Len = 27,
                };
            case kStatelessV1ID:
                // stateless v1
                [[fallthrough]];
            case kStatelessV2ID:
                // stateless v1 || v2
                return {
                    .From = rawSecret.length() - 22,
                    .Len = 22,
                };
            default:
                ythrow TSystemError() << "unexpected token pattern id: " << id;
        }
    }

    TString TYandexOAuth::Name() const {
        return "yandex-oauth";
    }

    NSecret::ESecretType TYandexOAuth::SecretType() const {
        return NSecret::ESecretType::YOAuth;
    }

    TMaybe<bool> TYandexOAuth::ForceValid() const {
        return Nothing();
    }
}
