#include "mds_sign.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    TVector<TString> TMdsSign::KeyPatterns() const {
        // value with uri contain some like:
        // https://avatars.mdst.yandex.net/get-assessortest/5476/secure-wm-test/optimize?
        // dynamic-watermark=somesecuretext&ts=1582160381&sign=48b35e4d9f49b7896b46ec7e9896605086e8ad7a71a25c6b39cca33d05dee635
        return {
            "sign",
        };
    }

    TVector<TString> TMdsSign::ValuePatterns() const {
        return {
            R"([0-9a-z]{64})",
        };
    }

    bool TMdsSign::IsSecret(size_t id, const TStringBuf secret) const {
        Y_UNUSED(id);
        return secret.size() == 64;
    }

    NSecret::TPos TMdsSign::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        return {
            .From = rawSecret.length() - 64,
            .Len = 64,
        };
    }

    TString TMdsSign::Name() const {
        return "mds-sign";
    }

    NSecret::ESecretType TMdsSign::SecretType() const {
        return NSecret::ESecretType::MdsSign;
    }

    TMaybe<bool> TMdsSign::ForceValid() const {
        return true;
    }
}

