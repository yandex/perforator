#include "tvm_user.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    namespace {
        size_t kGenericSafeLen = 16;
        size_t kMinSignLen = 200;

        bool isUserTicket(const TStringBuf rawTicket) {
            TStringBuf ticket;
            if (!rawTicket.AfterPrefix("3:user:", ticket)) {
                return false;
            }

            TStringBuf sign = ticket.After(':');
            if (sign.size() < kMinSignLen || NStringUtils::IsMasked(sign, kMinSignLen/2)) {
                return false;
            }

            TStringBuf body = ticket.Before(':');
            return NStringUtils::IsBase64Url(body) && NStringUtils::IsBase64Url(sign);
        }
    }

    TVector<TString> TTvmUserTicket::Patterns() const {
        return {
           R"(3\:user\:[\w\-\\]+\:([A-Za-z0-9\-]|\\?_){48,}[\w\-])",
        };
    }

    bool TTvmUserTicket::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);
        return isUserTicket(secret);
    }

    NSecret::TPos TTvmUserTicket::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        size_t signPos = rawSecret.find_last_of(':');
        if (signPos == TStringBuf::npos) {
            return {
                .From = kGenericSafeLen,
                .Len = rawSecret.length() - kGenericSafeLen,
            };
        }

        signPos++;
        return {
            .From = signPos,
            .Len = rawSecret.length() - signPos,
        };
    }

    TString TTvmUserTicket::Name() const {
        return "tvm-user-ticket";
    }

    NSecret::ESecretType TTvmUserTicket::SecretType() const {
        return NSecret::ESecretType::TVMTicket;
    }

    TMaybe<bool> TTvmUserTicket::ForceValid() const {
        return true;
    }
}
