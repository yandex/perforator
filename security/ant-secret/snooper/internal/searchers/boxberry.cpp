#include "boxberry.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    TVector<TString> TBoxberry::KeyPatterns() const {
        // https://st.yandex-team.ru/ANTSECRET-165
        return {
            "token",
        };
    }

    TVector<TString> TBoxberry::ValuePatterns() const {
        // uuid4
        return {
            R"([a-fA-F0-9]{32})",
        };
    }

    bool TBoxberry::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        return !NStringUtils::IsMasked(secret, secret.size());
    }

    NSecret::TPos TBoxberry::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        return {
            .From = 0,
            .Len = rawSecret.length(),
        };
    }

    TString TBoxberry::Name() const {
        return "boxberry";
    }

    NSecret::ESecretType TBoxberry::SecretType() const {
        return NSecret::ESecretType::BoxberryToken;
    }

    TMaybe<bool> TBoxberry::ForceValid() const {
        return false;
    }

    TVector<TString> TBoxberry::KvSeparators() const {
        return {
            R"(\s*: )",
            R"(\\?=)",
        };
    }

    TVector<TString> TBoxberry::QuotedSeparator() const {
        const TVector<TString> quotes = {
            R"(\\*")",
        };

        const TVector<TString> separators = {
            R"(\s*:\s*)",
            R"(\s*=\s*)",
        };

        TVector<TString> out;
        for (auto&& quote : quotes) {
            for (auto&& sep : separators) {
                out.push_back(quote+sep+quote);
            }
        }
        return out;
    }
}
