#include "yandex_session.h"

#include <security/ant-secret/internal/string_utils/common.h>

#include <util/string/vector.h>

namespace NSearchers {
    TVector<TString> TYandexSession::Patterns() const {
/*
Documentation: https://wiki.yandex-team.ru/passport/sessionid/#formatv3spodderzhkojjmultiavtorizacijj
Format: ver:ts.ttl.default_idx.authid.cookie_flags(.k:v)*(|uid.pwd_check.acc_flags(.k:v)*)+|keyid.rnd.signature
Examples:
    - 3:1552306280.5.0.1552217798723:Z8FsXw:1d.1|126480966.88482.2.2:88482|196108.640870.XXXXXXXXXXXXXXXXXXXX-6SznOg
    - 3:1552308886.5.0.1527855241846:UhFYTQ:88.1|1120000000001700.-1.400|1120000000026268.-1.0|1120000000102258.-1.0|1120000000081886.-1.400|1120000000038691.-1.0|1120000000123201.-1.0|134332.159643.XXXXXXXXXXXXXXXXXXXX-6SznOg
    - 3:1552308886.5.5.1527855241846:UhFYTQ:88.1|1120000000001700.20469790.2402.2:20469790|1120000000026268.9690762.2002.2:9690762|1120000000102258.20726003.2002.0:3.2:20726003|1120000000081886.21172253.2402.2:21172253|1120000000038691.24012992.2002.2:24012992|1120000000123201.24453645.2002.2:24453645|134332.183007.XXXXXXXXXXXXXXXXXXXX-6SznOg
    - 3:1551182683.5.0.1540542645970:LDh8YjAwYU0MBAAAuAYCKg:43.1|88994589.0.302|195482.741222.XXXXXXXXXXXXXXXXXXXX-6SznOg
    - 3:1635882127.5.0.1635882127771:IwABAAAAAAARgIGwuAYCKg:15.1|126480966.0.2|3:243045.152787.XXXXXXXXXXXXXXXXXXXX-6SznOg
    - 3:1653555178.5.0.1649862799022:PWqlXw:e.1.2:1.499:1|253917951.0.302|3:252899.416214.iaPip-XXXXXXXXZNq4BocomR8XM
*/
        return {
            //    |  ts  |ttl |   defId      |        authId               |   kv_block      |      users + user_kv                  |keyId   |rnd | sig
            {R"(3\:\d{10}\.\d+\.[0-9a-fA-F]{1,2}\.(\d+\:[\w\-]+\:[0-9a-fA-F]+|\d+)(\.[\w:%\-]+)*\|(\d+\.[\d\-]+\.\d+(\.[\w:%\-]+)*\|)+[\w:%\-]+\.\d+\.[\w\-]{27})"},
        };
    }

    bool TYandexSession::IsSecret(size_t id, TStringBuf secret) const {
        Y_UNUSED(id);
        return NStringUtils::IsBase64Url(secret.RAfter('.'));
    }

    NSecret::TPos TYandexSession::MaskSecret(size_t id, const TStringBuf rawSecret) const {
        Y_UNUSED(id);

        return {
            .From = rawSecret.length() - 27,
            .Len = 27,
        };
    }

    TString TYandexSession::Name() const {
        return "yandex-session";
    }

    NSecret::ESecretType TYandexSession::SecretType() const {
        return NSecret::ESecretType::YSession;
    }

    TMaybe<bool> TYandexSession::ForceValid() const {
        return Nothing();
    }
}
