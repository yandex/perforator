#pragma once

#include "searcher.h"
#include "secret.h"

#include <security/ant-secret/snooper/internal/ctx.h>
#include <util/system/mutex.h>

namespace NSnooper {
    class TSnooper {
    public:
        TSnooper() = default;

        THolder<TSearcher> Searcher(TSecretTypes neededSecrets = NSecret::ESecretType::All);

        TSearcher* NewSearcher(TSecretTypes neededSecrets = NSecret::ESecretType::All);

        THolder<TProtoSearcher> ProtoSearcher(TSecretTypes neededSecrets = NSecret::ESecretType::All);

    protected:
        NSnooperInt::TContext ctx;
    };

}
