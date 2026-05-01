#include "gitlab.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    namespace {
        enum EPattern {
            kPersonalTokenID,
            kPipelineTriggerTokenID,
            kRunnerRegistrationTokenID,
            kCICDTokenID,
        };

        inline size_t tokenMaskLen(size_t id) {
            switch (id) {
                case kPersonalTokenID:
                    return 20;
                case kPipelineTriggerTokenID:
                    return 40;
                case kRunnerRegistrationTokenID:
                    return 20;
                case kCICDTokenID:
                    return 20;
                default:
                    return 0;
            }
        }
    }

    TVector<TString> TGitlab::Patterns() const {
        return {
            // Personal Access Token
            R"(glpat-[0-9a-zA-Z\-\_]{20})",
            // Pipeline Trigger Token
            R"(glptt-[0-9a-f]{40})",
            // Runner Registration Token
            R"(GR1348941[0-9a-zA-Z\-\_]{20})",
            // CI/CD Job Token
            R"(glcbt-[0-9a-zA-Z]{1,5}_[0-9a-zA-Z_-]{20})",
        };
    }

    bool TGitlab::IsSecret(size_t id, const TStringBuf token) const {
        size_t maskLen = tokenMaskLen(id);
        if (maskLen <= 0) {
            ythrow TSystemError() << "unexpected token pattern id: " << id;
        }

        return !NStringUtils::IsMasked(token, maskLen);
    }

    NSecret::TPos TGitlab::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        size_t maskLen = tokenMaskLen(id);
        if (maskLen <= 0) {
            ythrow TSystemError() << "unexpected token pattern id: " << id;
        }

        return {
            .From = rawSecret.length() - maskLen,
            .Len = maskLen,
        };
    }

    TString TGitlab::Name() const {
        return "gitlab";
    }

    NSecret::ESecretType TGitlab::SecretType() const {
        return NSecret::ESecretType::Gitlab;
    }

    TMaybe<bool> TGitlab::ForceValid() const {
        return false;
    }
}
