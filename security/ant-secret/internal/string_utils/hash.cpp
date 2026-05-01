#include "hash.h"

#include <contrib/libs/openssl/include/openssl/sha.h>

#include <util/generic/string.h>

namespace NStringUtils {
    TString Sha1(const char* data, size_t len) {
        unsigned char digest[SHA_DIGEST_LENGTH] = {0};
        SHA1((const unsigned char*)data, len, digest);
        static const char hex[] = "0123456789abcdef";
        char buf[SHA_DIGEST_LENGTH * 2 + 1] = {0};
        for (int i = 0; i < SHA_DIGEST_LENGTH; ++i) {
            buf[i + i] = hex[digest[i] >> 4];
            buf[i + i + 1] = hex[digest[i] & 0x0f];
        }

        return TString(buf);
    }

    TString Sha1(const TStringBuf data) {
        return Sha1(data.data(), data.size());
    }

    TString Sha1(const TString& data) {
        return Sha1(data.data(), data.size());
    }

}
