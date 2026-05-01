#include "masker.h"

namespace NSnooperInt::NMask {
    namespace {
        constexpr char kMaskChar = 'X';
        constexpr TStringBuf kDevApiKeyMask = "****-****-****";

        void MaskSecret(TString& data, const NSecret::TSecret& secret) {
            switch (secret.Type) {
            case NSecret::ESecretType::DevApiKey:
                if (secret.MaskPos.Len == kDevApiKeyMask.size()) {
                    data.replace(secret.MaskPos.From, secret.MaskPos.Len, kDevApiKeyMask);
                    break;
                }

                [[fallthrough]];
            default:
                std::fill(data.begin() + secret.MaskPos.From, data.begin() + secret.MaskPos.From + secret.MaskPos.Len, kMaskChar);
            }
        }

        void MaskSecret(TArrayRef<char> data, const NSecret::TSecret& secret) {
            switch (secret.Type) {
            case NSecret::ESecretType::DevApiKey:
                if (secret.MaskPos.Len == kDevApiKeyMask.size()) {
                    std::memcpy(data.begin() + secret.MaskPos.From, kDevApiKeyMask.data(), kDevApiKeyMask.size());
                    break;
                }

                [[fallthrough]];
            default:
                std::fill(data.begin() + secret.MaskPos.From, data.begin() + secret.MaskPos.From + secret.MaskPos.Len, kMaskChar);
            }
        }
    }

    void MaskSecrets(TString& data, const NSecret::TSecretList& secrets) {
        for (auto&& secret : secrets) {
            MaskSecret(data, secret);
        }
    }

    void MaskSecrets(TArrayRef<char> data, const NSecret::TSecretList& secrets) {
        for (auto&& secret : secrets) {
            MaskSecret(data, secret);
        }
    }

}
