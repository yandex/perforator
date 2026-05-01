#pragma once

#include "utils.h"

#include <security/ant-secret/snooper/internal/ctx.h>
#include <security/ant-secret/snooper/internal/searchers/interface.h>

#include <util/generic/string.h>
#include <util/generic/strbuf.h>
#include <util/generic/vector.h>

namespace NSearchers {
    class TPrefixed: public ISearcher {
    public:
        explicit TPrefixed(NSnooperInt::TContext ctx)
            : ctx(std::move(ctx))
        {};

        bool Compile() override;

        bool SearchInto(const TSearchRequest& req, NSecret::TSecretList& out) const override;

        bool SearchValidatedInto(const TSearchRequest& req, NSecret::TSecretList& out) const override;

        TVector<TString> SearchPatterns() const override;

    protected:
        virtual TVector<TString> KeyPatterns() const = 0;

        virtual TVector<TString> ValuePatterns() const = 0;

        inline virtual NSecret::ESecretType SecretType() const = 0;

        virtual bool IsSecret(size_t id, TStringBuf secret) const = 0;

        virtual NSecret::TPos MaskSecret(size_t id, const TStringBuf rawSecret) const = 0;

        virtual TVector<TString> KvSeparators() const;

        virtual TVector<TString> QuotedSeparator() const;

    protected:
        NSnooperInt::TContext ctx;
        TVector<TString> keyPatterns;
        TVector<TPattern> valuePatterns;
    };

}
