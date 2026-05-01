#include "jwt.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <library/cpp/string_utils/base64/base64.h>
#include <util/string/split.h>
#include <util/generic/vector.h>


namespace NJwt {
    namespace {
        TString lastBase64Char(TStringBuf s) {
            size_t padding = 4 - s.length() % 4;
            if (padding == 4) {
                return Base64Decode(s.Last(4));
            }

            return Base64Decode(TString(s.Last(4 - padding)) + TString(padding, '='));
        }

        bool isLastCharInBase64(TStringBuf s, TStringBuf::char_type ch) {
            return lastBase64Char(s).EndsWith(ch);
        }
    }

    bool IsHeader(TStringBuf rawHeader) {
        auto header = Base64DecodeUneven(rawHeader);
        if (!header.StartsWith('{') && !header.EndsWith('}')) {
            // not a JSON string
            return false;
        }

        // algo is the only one REQUIRED parameter for JWT/JWS
        return header.Contains("\"alg\"");
    }

    bool IsPayload(TStringBuf rawPayload) {
        if (!rawPayload.StartsWith("ey")) {
            return false;
        }

        return isLastCharInBase64(rawPayload, '}');
    }

    bool IsToken(TStringBuf token) {
        auto parts = StringSplitter(token).Split('.').Limit(3).ToList<TStringBuf>();
        if (parts.size() != 3 ) {
            return false;
        }

        if (NStringUtils::IsMasked(parts[2], parts[2].length()/2)) {
            return false;
        }

        if (!IsPayload(parts[1])) {
            return false;
        }

        if (!IsHeader(parts[0])) {
            return false;
        }

        return true;
    }
}
