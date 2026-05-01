#pragma once

#include "result.h"

#include <library/cpp/cache/thread_safe_cache.h>
#include <util/generic/strbuf.h>
#include <util/generic/maybe.h>

namespace NValidation {
    class TValidatorCache; // forward declaration

    class IValidator {
    public:
        virtual ~IValidator() = default;

        virtual TMaybe<TResult>
        Call(const TStringBuf type, const TStringBuf token) = 0;

        virtual std::pair<bool, bool>
        CallSsl(const TString& serial, const TVector<TString>& chain, bool isClient, bool isServer) = 0;

        virtual TMaybe<bool>
        CallIsKnown(const TStringBuf type, const TStringBuf secret) = 0;
    };

    class TValidator: public IValidator {
    public:
        TValidator();

        explicit TValidator(const TString& host);

        TValidator(const TString& host, const TString& ip);

        TMaybe<TResult>
        Call(const TStringBuf type, const TStringBuf token) override;

        TResult*
        CallBypassCache(const TStringBuf type, const TStringBuf token);

        std::pair<bool, bool>
        CallSsl(const TString& serial, const TVector<TString>& chain, bool isClient, bool isServer) override;

        TMaybe<bool>
        CallIsKnown(const TStringBuf vcs, const TStringBuf secret) override;

        void SkipKnown(bool skip);
        void BetaFeatures(bool enabled);
        void WithAuthToken(const TString& authToken);

    protected:
        bool skipKnown = false;
        bool betaFeatures = false;
        TString baseUrl{};
        TString validateUrl{};
        TString addr{};
        TString authToken{};
        THolder<TValidatorCache> cache;
    };

    using TValidatorCacheImpl = TThreadSafeCache<TString, TResult, const TStringBuf, const TStringBuf>;

    class TValidatorCache: public TValidatorCacheImpl::ICallbacks {
    public:
        explicit TValidatorCache(TValidator& validator, size_t cacheSize = 1000);

        TMaybe<TResult>
        Get(const TStringBuf type, const TStringBuf token);

    private:
        //  TThreadSafeCache interface
        TResult* CreateObject(const TStringBuf type, const TStringBuf token) const final;

        TString GetKey(const TStringBuf type, const TStringBuf token) const final;

    private:
        TValidator& validator;
        TValidatorCacheImpl cache;
    };

}
