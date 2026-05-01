#include "yf_token.h"

#include <security/ant-secret/internal/string_utils/common.h>
#include <security/ant-secret/internal/base62/base62.h>

#include <util/generic/maybe.h>

namespace NSearchers {
    namespace {
        constexpr size_t kPrefixLen = 13; // yf + env + _ + 5-letter kind + _ + v + digit + _
        constexpr size_t kEntropyLen = 44;
        constexpr size_t kChecksumLen = 6;
    }

    TVector<TString> TYFToken::Patterns() const {
        return {
            // 44 entropy + 6 CRC (base62)
            R"(yf[0-9]_[a-zA-Z]{5}_v[0-9]_[A-Za-z0-9]{44}[A-Za-z0-9]{6})",
        };
    }

    bool TYFToken::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        const TStringBuf entropy = secret.SubString(kPrefixLen, kEntropyLen);
        if (NStringUtils::IsMasked(entropy, entropy.size())) {
            return false;
        }

        return NBase62::IsToken(secret, 0, kChecksumLen);
    }

    NSecret::TPos TYFToken::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);
        Y_UNUSED(rawSecret);

        return {
            .From = kPrefixLen,
            .Len = kEntropyLen,
        };
    }

    TString TYFToken::Name() const {
        return "yf-token";
    }

    NSecret::ESecretType TYFToken::SecretType() const {
        return NSecret::ESecretType::YFToken;
    }

    TMaybe<bool> TYFToken::ForceValid() const {
        return Nothing();
    }
}
