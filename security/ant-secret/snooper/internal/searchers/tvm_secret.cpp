#include "tvm_secret.h"

#include <security/ant-secret/internal/tvm/tvm.h>

namespace NSearchers {
    TVector<TString> TTvmSecret::Patterns() const {
        return {
           R"(tvm-[A-Za-z0-9_-]{27,40})",
        };
    }

    bool TTvmSecret::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);
        return  NTVM::IsSecret(secret);
    }

    NSecret::TPos TTvmSecret::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        size_t toSkip = 0;
        if (rawSecret.starts_with("tvm-")) {
            toSkip = 4;
        }

        return {
            .From = toSkip,
            .Len = rawSecret.length() - toSkip,
        };
    }

    TString TTvmSecret::Name() const {
        return "tvm-secret";
    }

    NSecret::ESecretType TTvmSecret::SecretType() const {
        return NSecret::ESecretType::TVMSecret;
    }

    TMaybe<bool> TTvmSecret::ForceValid() const {
       return Nothing();
    }
}
