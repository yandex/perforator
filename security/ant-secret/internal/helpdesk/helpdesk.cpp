#include "helpdesk.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <library/cpp/digest/old_crc/crc.h>

#include <util/string/split.h>

namespace NHelpDesk {
    constexpr size_t kMinRoboPassLen = 60;
    constexpr size_t kMinRoboPassV2Len = 38;

    bool IsRobotPassword(const TStringBuf token) {
        if (token.size() < kMinRoboPassLen) {
            return false;
        }

        auto parts = StringSplitter(token).Split('_').ToList<TStringBuf>();
        if (parts.size() != 4) {
            return false;
        }

        TStringBuf secretUUID = parts[1];
        if (!NStringUtils::IsBase64Even(secretUUID)) {
            return false;
        }

        TStringBuf rawCRC32 = parts[3];
        ui64 expectedChecksum = FromString<ui64>(rawCRC32);
        if (expectedChecksum == 0 || expectedChecksum > 4294967295) {
            return false;
        }

        ui64 actualChecksum = crc32(token.data(), token.size() - rawCRC32.size() - 1);
        return actualChecksum == expectedChecksum;
    }

    bool IsRobotPasswordV2(const TStringBuf token) {
        if (token.size() < kMinRoboPassV2Len) {
            return false;
        }

        auto parts = StringSplitter(token).Split('_').ToList<TStringBuf>();
        if (parts.size() != 4) {
            return false;
        }

        TStringBuf uid = parts[1];
        if (!NStringUtils::IsBase64(uid)) {
            return false;
        }

        TStringBuf rawCRC32 = parts[3];
        ui64 expectedChecksum = FromString<ui64>(rawCRC32);
        if (expectedChecksum == 0 || expectedChecksum > 4294967295) {
            return false;
        }

        ui64 actualChecksum = crc32(token.data(), token.size() - rawCRC32.size() - 1);
        return actualChecksum == expectedChecksum;
    }
}
