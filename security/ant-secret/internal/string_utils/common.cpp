#include "common.h"

namespace NStringUtils {
    bool IsHash(const TStringBuf data) {
        if (data.size() % 2 != 0) {
            // hash string must be multiple of 2
            return false;
        }

        bool haveLower = false;
        bool haveDigit = false;
        for (const auto& ch : data) {
            if ('0' <= ch && ch <= '9') {
                haveDigit = true;
            } else if ('a' <= ch && ch <= 'f') {
                haveLower = true;
            } else {
                return false;
            }
        }

        // by statistic hash probably contain lower chars and digit
        return haveLower && haveDigit;
    }

    bool IsBase64(const TStringBuf data) {
        bool haveUpper = false;
        bool haveLower = false;
        for (const auto& ch : data) {
            if ('0' <= ch && ch <= '9') {
                // pass
            } else if ('a' <= ch && ch <= 'z') {
                haveLower = true;
            } else if ('A' <= ch && ch <= 'Z') {
                haveUpper = true;
            } else if (ch == '+' || ch == '/' | ch == '=') {
                // pass
            } else {
                return false;
            }
        }

        // by statistic base64 probably contain lower/upper char and digit
        return haveUpper && haveLower;
    }

    bool IsBase64Even(const TStringBuf data) {
        if (data.size() % 4 != 0) {
            // Base64 string must be multiple of 4
            return false;
        }

        return IsBase64(data);
    }

    bool IsBase64Url(const TStringBuf data) {
        bool haveUpper = false;
        bool haveLower = false;
        for (const auto& ch : data) {
            if ('0' <= ch && ch <= '9') {
                // pass
            } else if ('a' <= ch && ch <= 'z') {
                haveLower = true;
            } else if ('A' <= ch && ch <= 'Z') {
                haveUpper = true;
            } else if (ch == '*' || ch == '-' | ch == '_') {
                // pass
            } else {
                return false;
            }
        }

        // by statistic base64 probably contain lower and upper chars
        return haveUpper && haveLower;
    }

    bool IsBase64UrlRaw(const TStringBuf data) {
        bool haveUpper = false;
        bool haveLower = false;
        for (const auto& ch : data) {
            if ('0' <= ch && ch <= '9') {
                // pass
            } else if ('a' <= ch && ch <= 'z') {
                haveLower = true;
            } else if ('A' <= ch && ch <= 'Z') {
                haveUpper = true;
            } else if (ch == '*' || ch == '-' | ch == '_' || ch == '=') {
                // pass
            } else {
                return false;
            }
        }

        // by statistic base64 probably contain lower and upper chars
        return haveUpper && haveLower;
    }

    bool IsBase64UrlEven(const TStringBuf data) {
        if (data.size() % 4 != 0) {
            // Base64 string must be multiple of 4
            return false;
        }

        return IsBase64UrlRaw(data);
    }

    bool IsMasked(const TStringBuf data, size_t maskLen) {
        size_t start = data.size()-maskLen;
        for (size_t i = ++start; i < data.size(); ++i) {
            if (data[i-1] != data[i]) {
                return false;
            }
        }

        return true;
    }

    bool IsValidUTF8(const TStringBuf in) {
        const uint8_t* data = reinterpret_cast<const uint8_t*>(in.data());
        size_t len = in.size();

        for (size_t i = 0; i < len; ) {
            uint8_t byte = data[i];
            if (byte <= 0x7F) {
                // ASCII
                i += 1;
                continue;
            }

            if ((byte >> 5) == 0x6 && i + 1 < len &&
                    (data[i + 1] & 0xC0) == 0x80) {

                // 2-byte sequence
                uint32_t codepoint = ((byte & 0x1F) << 6) | (data[i + 1] & 0x3F);
                if (codepoint < 0x80) {
                    // overlong encoding
                    return false;
                }

                i += 2;
                continue;
            }

            if ((byte >> 4) == 0xE && i + 2 < len &&
                    (data[i + 1] & 0xC0) == 0x80 &&
                    (data[i + 2] & 0xC0) == 0x80) {

                // 3-byte sequence
                uint32_t codepoint = ((byte & 0x0F) << 12) | ((data[i + 1] & 0x3F) << 6) | (data[i + 2] & 0x3F);
                if (codepoint < 0x800 || (codepoint >= 0xD800 && codepoint <= 0xDFFF)) {
                    // overlong or surrogate
                    return false;
                }

                i += 3;
                continue;
            }

            if ((byte >> 3) == 0x1E && i + 3 < len &&
                    (data[i + 1] & 0xC0) == 0x80 &&
                    (data[i + 2] & 0xC0) == 0x80 &&
                    (data[i + 3] & 0xC0) == 0x80) {

                // 4-byte sequence
                uint32_t codepoint = ((byte & 0x07) << 18) | ((data[i + 1] & 0x3F) << 12) | ((data[i + 2] & 0x3F) << 6) | (data[i + 3] & 0x3F);
                if (codepoint < 0x10000 || codepoint > 0x10FFFF) {
                    // overlong or out of Unicode range
                    return false;
                }

                i += 4;
                continue;
            }


            // invalid start byte or insufficient continuation bytes
            return false;
        }

        return true;
    }
}
