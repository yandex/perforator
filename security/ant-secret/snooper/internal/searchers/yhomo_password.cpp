#include "yhomo_password.h"

#include <security/ant-secret/internal/yhomo/yhomo.h>

namespace NSearchers {
    TVector<TString> TYHomoPassword::Patterns() const {
        return {
            R"(yh0:[A-Za-z0-9_\-\\]{8,}:[A-Za-z0-9~!@#$%^*()_+{}?><;,.\\]{8,}:[a-f0-9]{8})",
        };
    }

    bool TYHomoPassword::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        return NYHomo::IsPassword(secret);
    }

    NSecret::TPos TYHomoPassword::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        auto passLen = rawSecret.After(':').After(':').size();
        return {
            .From = rawSecret.length() - passLen,
            .Len = passLen,
        };
    }

    TString TYHomoPassword::Name() const {
        return "yhomo";
    }

    NSecret::ESecretType TYHomoPassword::SecretType() const {
        return NSecret::ESecretType::YHomoPassword;
    }

    TMaybe<bool> TYHomoPassword::ForceValid() const {
        return Nothing();
    }
}
