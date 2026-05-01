#include "yrobo_password.h"

#include <security/ant-secret/internal/helpdesk/helpdesk.h>

namespace NSearchers {
    TVector<TString> TYRoboPassword::Patterns() const {
        return {
            R"(yv\\?_[A-Za-z0-9+/\\]{40,}(?:={0,2})?\\?_[QWERTYUPASDFGHJKLZXCVBNMqwertyuiopasdfghjkzxcvbnm1-9]{15,}\\?_[0-9]{8,11})",
        };
    }

    bool TYRoboPassword::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        return NHelpDesk::IsRobotPassword(secret);
    }

    NSecret::TPos TYRoboPassword::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        auto passLen = rawSecret.After('_').After('_').size();
        return {
            .From = rawSecret.length() - passLen,
            .Len = passLen,
        };
    }

    TString TYRoboPassword::Name() const {
        return "yrobo";
    }

    NSecret::ESecretType TYRoboPassword::SecretType() const {
        return NSecret::ESecretType::YRoboPassword;
    }

    TMaybe<bool> TYRoboPassword::ForceValid() const {
        return Nothing();
    }
}
