#pragma once

#include "ctx.h"
#include <util/generic/strbuf.h>
#include <contrib/libs/re2/re2/set.h>
#include <security/ant-secret/internal/validation/validator.h>
#include <security/ant-secret/snooper/internal/secret/secret.h>
#include <security/ant-secret/snooper/internal/searchers/interface.h>

namespace NSnooperInt {

    struct TSearcherInfo {
        TSearcherInfo(size_t searcherID, size_t patternID)
            : searcherID(searcherID)
            , patternID(patternID) {}

        size_t searcherID;
        size_t patternID;
    };

    class TSearchersStore {
    public:
        TSearchersStore(NSnooperInt::TContext ctx, NSecret::TSecretTypes neededSecrets = NSecret::ESecretType::All);

        NSecret::TSecretList Search(TStringBuf data, bool validOnly) const;
    protected:
        TVector<THolder<NSearchers::ISearcher>> searchers;
        re2::RE2::Set preMatcher;
        TVector<TSearcherInfo> searchersInfo;
    };

}
