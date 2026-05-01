#include "tvm_service.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>


namespace NSearchers {
    namespace {
        size_t kGenericSafeLen = 16;
        size_t kMinSignLen = 200;

        bool isServiceTicket(const TStringBuf rawTicket) {
            TStringBuf ticket;
            if (!rawTicket.AfterPrefix("3:serv:", ticket)) {
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

    TVector<TString> TTvmServiceTicket::Patterns() const {
        return {
            R"(3\:serv\:[\w\-\\]+\:([A-Za-z0-9\-]|\\?_){48,}[\w\-])",
        };
    }

    bool TTvmServiceTicket::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);
        return isServiceTicket(secret);
    }

    NSecret::TPos TTvmServiceTicket::MaskSecret(size_t id, const TStringBuf rawSecret) const {
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

    TString TTvmServiceTicket::Name() const {
        return "tvm-service-ticket";
    }

    NSecret::ESecretType TTvmServiceTicket::SecretType() const {
        return NSecret::ESecretType::TVMTicket;
    }

    TMaybe<bool> TTvmServiceTicket::ForceValid() const {
        return true;
    }
}
