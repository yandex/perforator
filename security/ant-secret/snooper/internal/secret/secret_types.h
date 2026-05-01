#pragma once

#include <util/system/types.h>
#include <util/generic/flags.h>

namespace NSecret {
    enum class ESecretType : ui32 {
        Unknown = 0,
        YOAuth = 1 << 0,
        YSession = 1 << 1,
        TVMTicket = 1 << 2,
        S3Presign = 1 << 3,
        MdsSign = 1 << 4,
        YCApiKey = 1 << 5,
        YCCookie = 1 << 6,
        YCToken = 1 << 7,
        YCStaticCred = 1 << 8,
        JwtToken = 1 << 9,
        Gitlab = 1 << 10,
        YRoboPassword = 1 << 11,
        YHomoPassword = 1 << 12,
        UrlPassword = 1 << 13,
        MarketPull = 1 << 14,
        MsCookie = 1 << 15,
        DevApiKey = 1 << 16,
        BoxberryToken = 1 << 17,
        GeoSharingToken = 1 << 18,
        TVMSecret = 1 << 19,
        S3SecretKey = 1 << 20,
        YFToken = 1 << 21,
        YCRefreshToken = 1 << 22,
        YCOAuthSecret = 1 << 23,

        TrulySecrets =
            Gitlab
            | MarketPull
            | MsCookie
            | S3SecretKey
            | TVMSecret
            | TVMTicket
            | YCApiKey
            | YCCookie
            | YCOAuthSecret
            | YCRefreshToken
            | YCStaticCred
            | YCToken
            | YHomoPassword
            | YOAuth
            | YRoboPassword
            | YSession
            | YFToken,

        All =
            TrulySecrets

            | BoxberryToken
            | DevApiKey
            | GeoSharingToken
            | JwtToken
            | S3Presign
            | UrlPassword,

        AllWMds = All | MdsSign,
    };

    Y_DECLARE_FLAGS(TSecretTypes, ESecretType);

    Y_DECLARE_OPERATORS_FOR_FLAGS(TSecretTypes);
}
