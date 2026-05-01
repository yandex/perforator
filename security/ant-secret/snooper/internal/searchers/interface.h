#pragma once

#include <security/ant-secret/snooper/internal/secret/secret.h>
#include <contrib/libs/re2/re2/re2.h>

#include <util/generic/vector.h>
#include <util/generic/strbuf.h>
#include <util/generic/maybe.h>

namespace NSearchers {
    struct TSearchRequest {
        TStringBuf data;
        size_t keyId;
    };

    class ISearcher {
    public:
        virtual ~ISearcher() = default;

        virtual bool Compile() = 0;

        virtual bool SearchInto(const TSearchRequest& req, NSecret::TSecretList& out) const = 0;

        virtual bool SearchValidatedInto(const TSearchRequest& req, NSecret::TSecretList& out) const = 0;

        virtual TVector<TString> SearchPatterns() const = 0;

        inline virtual TString Name() const = 0;

        inline virtual TMaybe<bool> ForceValid() const = 0;
    };

    struct TPattern {
        size_t patternID;
        THolder<re2::RE2> re;
    };
}
