#include "yrobo_password_v2.h"

#include <security/ant-secret/internal/helpdesk/helpdesk.h>

namespace NSearchers {
    TVector<TString> TYRoboPasswordV2::Patterns() const {
        return {
            R"(yv\\?_[A-Za-z0-9+/\\]{11,12}\\?_[QWERTYUPASDFGHJKLZXCVBNMqwertyuiopasdfghjkzxcvbnm1-9]{15,}\\?_[0-9]{8,11})",
        };
    }

    bool TYRoboPasswordV2::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);

        return NHelpDesk::IsRobotPasswordV2(secret);
    }

    NSecret::TPos TYRoboPasswordV2::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        auto passLen = rawSecret.After('_').After('_').size();
        return {
            .From = rawSecret.length() - passLen,
            .Len = passLen,
        };
    }

    TString TYRoboPasswordV2::Name() const {
        return "yrobo";
    }

    NSecret::ESecretType TYRoboPasswordV2::SecretType() const {
        return NSecret::ESecretType::YRoboPassword;
    }

    TMaybe<bool> TYRoboPasswordV2::ForceValid() const {
        return Nothing();
    }
}
