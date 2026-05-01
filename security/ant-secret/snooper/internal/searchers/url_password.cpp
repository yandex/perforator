#include "url_password.h"

#include <utility>

#include <util/string/vector.h>
#include <security/ant-secret/internal/string_utils/common.h>


namespace NSearchers {
    namespace {
        std::pair<size_t, size_t> passwordPos(const TStringBuf wholeUrl) {
            size_t from = wholeUrl.find('/');
            if (from == TStringBuf::npos) {
                return {0, 0};
            }
            from++;

            from = wholeUrl.find('/', from);
            if (from == TStringBuf::npos) {
                return {0, 0};
            }
            from++;

            from = wholeUrl.find(':', from);
            if (from == TStringBuf::npos) {
                return {0, 0};
            }
            from++;

            size_t to = wholeUrl.find('@', from);
            if (to == TStringBuf::npos) {
                return {0, 0};
            }

            return {from, to};
        }
    }

    TVector<TString> TUrlPassword::Patterns() const {
        return {
            R"([a-z0-9]{3,20}://[a-zA-Z0-9_\-\\]{3,20}:[^$][^:@/\s]{6,40}@[a-z0-9.\-]{10,})",
        };
    }

    bool TUrlPassword::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        auto [from, to] = passwordPos(secret);
        if (from == 0 || to == 0) {
            return false;
        }

        size_t len = to - from;
        return !NStringUtils::IsMasked(secret.SubString(from, len), len/2);
    }

    NSecret::TPos TUrlPassword::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        auto [from, to] = passwordPos(rawSecret);
        if (from == 0 || to == 0) {
            return {
                .From = 0,
                .Len = rawSecret.length(),
            };
        }

        return {
            .From = from,
            .Len = to - from,
        };
    }

    TString TUrlPassword::Name() const {
        return "url-password";
    }

    NSecret::ESecretType TUrlPassword::SecretType() const {
        return NSecret::ESecretType::UrlPassword;
    }

    TMaybe<bool> TUrlPassword::ForceValid() const {
        return false;
    }

    bool TUrlPassword::Uglified() const {
        return false;
    }
}
