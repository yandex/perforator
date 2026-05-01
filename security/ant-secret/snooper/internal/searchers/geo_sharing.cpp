#include "geo_sharing.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/split.h>
#include <util/system/byteorder.h>
#include <library/cpp/digest/crc32c/crc32c.h>
#include <library/cpp/string_utils/base64/base64.h>

namespace NSearchers {
    namespace {
        constexpr size_t kPrefixLen = 4;
    }

    TVector<TString> TGeoSharing::Patterns() const {
        return {
            // https://st.yandex-team.ru/ANTSECRET-170
            R"(lst_[A-Za-z0-9_-]{66,80})",
        };
    }

    bool TGeoSharing::IsSecret(size_t id, const TStringBuf token) const {
        Y_UNUSED(id);

        if (token.size() < 66) {
            return false;
        }

        if (NStringUtils::IsMasked(token, token.size() / 2)) {
            return false;
        }

        TString decoded = Base64DecodeUneven(TStringBuf(token).Skip(kPrefixLen));
        if (decoded.size() <= sizeof(ui32)) {
            return false;
        }

        ui32 actualChecksum = *reinterpret_cast<const ui32*>(decoded.data() + decoded.size() - sizeof(ui32));
        ui32 expectedChecksum= HostToLittle(Crc32c(decoded.data(), decoded.size() - sizeof(ui32)));
        return actualChecksum == expectedChecksum;
    }

    NSecret::TPos TGeoSharing::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id, rawSecret);

        return {
            .From = kPrefixLen,
            .Len = rawSecret.size() - kPrefixLen,
        };
    }

    TString TGeoSharing::Name() const {
        return "geo-sharing";
    }

    NSecret::ESecretType TGeoSharing::SecretType() const {
        return NSecret::ESecretType::GeoSharingToken;
    }

    TMaybe<bool> TGeoSharing::ForceValid() const {
        return Nothing();
    }
}
