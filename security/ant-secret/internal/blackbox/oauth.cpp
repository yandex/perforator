#include "oauth.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <library/cpp/string_utils/base64/base64.h>
#include <library/cpp/digest/old_crc/crc.h>

namespace NBlackBox {
    namespace {
        constexpr size_t kV2TokenLen = 39; // base64url(shard, uid, clientid, random)
        constexpr size_t kV3TokenMinLen = 40; // y(env)_base64url(shard, uid, clientid, token_id, random, crc32)
        constexpr size_t kMinBodyLen = 28;
        constexpr ui8 kMaxShard = 19;
        constexpr ui8 kMaskLen = 20;
        constexpr ui32 kMaxClientId = 1e7;

        bool IsEmbeddedBody(const TStringBuf body) {
            ui8 shard = body[0];
            if (shard == 0 || shard > kMaxShard) {
                return false;
            }

            ui64 uid = NStringUtils::FromBytesMsb<ui64>(body, 1);
            if (uid == 0) {
                return false;
            }


            ui32 clientId = NStringUtils::FromBytesMsb<ui32>(body, 9);
            if (clientId == 0 || clientId > kMaxClientId) {
                return false;
            }

            return true;
        }

        bool IsValidCrc(const TStringBuf body) {
            ui32 expectedChecksum = NStringUtils::FromBytesMsb<ui32>(body, body.size() - 4);
            // It's important to use crc32, not crc32c
            ui32 actualChecksum = crc32(body.data(), body.size() - 4);
            return actualChecksum == expectedChecksum;
        }
    }

    bool IsEmbeddedV2Token(const TStringBuf token) {
        if (token.size() != kV2TokenLen) {
            return false;
        }

        if (NStringUtils::IsMasked(token, kMaskLen)) {
            return false;
        }

        TString decoded = Base64DecodeUneven(token);
        if (decoded.size() < kMinBodyLen) {
            return false;
        }

        return IsEmbeddedBody(decoded);
    }

    bool IsEmbeddedV3Token(const TStringBuf token) {
        if (token.size() < kV3TokenMinLen) {
            return false;
        }

        if (token[0] != 'y' || token[2] != '_') {
            return false;
        }

        if (NStringUtils::IsMasked(token, kMaskLen)) {
            return false;
        }

        TString decoded = Base64DecodeUneven(TStringBuf(token).Skip(3));
        if (decoded.size() < kMinBodyLen) {
            return false;
        }

        if (token[3] != '_') {
            // '_' magic mean proto body: https://st.yandex-team.ru/PASSPINFRA-3339#6750a46ec1899d064972c10c
            if (!IsEmbeddedBody(decoded)) {
                return false;
            }
        }

        return IsValidCrc(decoded);
    }
}
