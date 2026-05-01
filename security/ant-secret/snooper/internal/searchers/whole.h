#pragma once

#include "utils.h"

#include <security/ant-secret/snooper/internal/ctx.h>
#include <security/ant-secret/snooper/internal/searchers/interface.h>

#include <contrib/libs/re2/re2/re2.h>

#include <util/generic/string.h>
#include <util/generic/strbuf.h>
#include <util/generic/vector.h>

namespace NSearchers {
    class TWhole: public ISearcher {
    public:
        explicit TWhole(NSnooperInt::TContext ctx)
            : ctx(std::move(ctx))
        {};

        bool Compile() override;

        bool SearchInto(const TSearchRequest& req, NSecret::TSecretList& out) const override;

        bool SearchValidatedInto(const TSearchRequest& req, NSecret::TSecretList& out) const override;

        TVector<TString> SearchPatterns() const override;

    protected:
        virtual bool Uglified() const;

        virtual TVector<TString> Patterns() const = 0;

        inline virtual NSecret::ESecretType SecretType() const = 0;

        virtual bool IsSecret(size_t id, TStringBuf secret) const = 0;

        virtual NSecret::TPos MaskSecret(size_t id, const TStringBuf rawSecret) const = 0;

    protected:
        NSnooperInt::TContext ctx;
        TVector<TPattern> patternsInfo;
    };

}
