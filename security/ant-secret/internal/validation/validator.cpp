#include "validator.h"

#include <security/libs/cpp/log/log.h>

#include <library/cpp/neh/neh.h>
#include <library/cpp/neh/http_common.h>
#include <library/cpp/json/json_reader.h>
#include <library/cpp/json/json_writer.h>
#include <util/datetime/base.h>
#include <library/cpp/string_utils/quote/quote.h>
#include <util/string/builder.h>
#include <util/system/env.h>

#include <utility>

namespace NValidation {
    namespace {
        constexpr TDuration kHttpDeadline = TDuration::Seconds(3);
        constexpr TStringBuf kServiceSchema = "http://";
        constexpr TStringBuf kServiceHost = "ant.sec.yandex-team.ru";
        constexpr TStringBuf kServicePath = "/api/v1/";

        constexpr TStringBuf kSchemeFull = "full";
    }

    TValidator::TValidator()
            : TValidator(TString(),  TString())
    {
    }

    TValidator::TValidator(const TString& host)
        : TValidator(host,  TString())
    {
    }

    TValidator::TValidator(const TString& host, const TString& ip)
        : cache(new TValidatorCache(*this))
    {
        baseUrl = TString(kServiceSchema) + (!host.empty() ? host : kServiceHost) + kServicePath;
        validateUrl = baseUrl + "validate/";

        if (!ip.empty()) {
            addr = TString(kSchemeFull) + "://[" + ip + "]";
        } else if (!host.empty()) {
            addr = TString(kSchemeFull) + "://" +  host;
        } else {
            addr = TString(kSchemeFull) + "://" +  kServiceHost;
        }
    }

    TMaybe<TResult> TValidator::Call(const TStringBuf type, const TStringBuf token) {
        if (Y_UNLIKELY(!type)) {
            ythrow TSystemError() << "call remote validation with empty type";
        }

        return cache->Get(type, token);
    }

    void TValidator::SkipKnown(bool skip) {
        skipKnown = skip;
    }

    void TValidator::BetaFeatures(bool enabled) {
        betaFeatures = enabled;
    }

    void TValidator::WithAuthToken(const TString& token) {
        authToken = std::move(token);
    }

    TResult* TValidator::CallBypassCache(const TStringBuf type, const TStringBuf token) {
        auto req = NNeh::TMessage(validateUrl + type, "");
        TStringBuilder headers;

        if (!authToken.empty()) {
            headers << "X-Internal-Token: " << authToken << "\r\n";
        }

        if (betaFeatures) {
          headers << "X-With-Beta-Features: 1\r\n";
        }

        TStringBuilder body;
        body << "secret=" << UrlEscapeRet(token);
        if (skipKnown) {
            body << "&skip_known=true";
        }

        bool ok = NNeh::NHttp::MakeFullRequest(req, headers, body,
                                               "application/x-www-form-urlencoded",
                                               NNeh::NHttp::ERequestType::Post);
        if (Y_UNLIKELY(!ok)) {
            NSecurityHelpers::LogErr("Failed to make validation service request", "type", type);
            return nullptr;
        }

        req.Addr = addr;
        NNeh::TResponseRef resp;
        if (!NNeh::Request(req)->Wait(resp, kHttpDeadline)) {
            NSecurityHelpers::LogErr("Failed to call validation service", "type", type, "err", "timed out");
            return nullptr;
        }

        if (Y_UNLIKELY(!resp)) {
            NSecurityHelpers::LogErr("Failed to call validation service ", "type", type, "err", "empty response");
            return nullptr;
        }

        if (resp->IsError()) {
            NSecurityHelpers::LogErr("Failed to call validation service", "type", type, "err", resp->GetErrorText());
            return nullptr;
        }

        NJson::TJsonValue data;
        if (!ReadJsonTree(resp->Data, &data)) {
            NSecurityHelpers::LogErr("Failed to parse validation service response", "type", type);
            return nullptr;
        }

        return TResult::NewFromJson(data);
    }

    std::pair<bool, bool>
    TValidator::CallSsl(const TString& serial, const TVector<TString>& chain, bool isClient, bool isServer) {
        TStringStream body;
        NJson::TJsonWriter json(&body, false);
        json.OpenMap();
        json.Write("serial", serial);
        json.Write("client", isClient);
        json.Write("server", isServer);
        if (!chain.empty()) {
            json.OpenArray("chain");
            for (const auto& item : chain) {
                json.Write(item);
            }
            json.CloseArray();
        }
        json.CloseMap();
        json.Flush();

        auto req = NNeh::TMessage(validateUrl + "ssl", "");
        bool ok = NNeh::NHttp::MakeFullRequest(
            req, TString(),
            body.Str(), "application/json",
            NNeh::NHttp::ERequestType::Post
        );

        auto result = std::make_pair<bool, bool>(false, false);
        if (Y_UNLIKELY(!ok)) {
            NSecurityHelpers::LogErr("Failed to make SSL validation service request");
            return result;
        }

        req.Addr = addr;
        NNeh::TResponseRef resp;
        if (!NNeh::Request(req)->Wait(resp, kHttpDeadline)) {
            NSecurityHelpers::LogErr("Failed to call SSL validation service", "err", "timed out");
            return result;
        }

        if (Y_UNLIKELY(!resp)) {
            NSecurityHelpers::LogErr("Failed to call SSL validation service", "err", "empty response");
            return result;
        }

        if (resp->IsError()) {
            NSecurityHelpers::LogErr("Failed to call SSL validation service", "err", resp->GetErrorText());
            return result;
        }

        NJson::TJsonValue data;
        if (!ReadJsonTree(resp->Data, &data)) {
            NSecurityHelpers::LogErr("Failed to parse SSL validation service response");
            return result;
        }

        if (!data["ok"].GetBooleanSafe(false)) {
            return result;
        }

        result.first = data["valid"].GetBooleanSafe(false);
        result.second = data["revoked"].GetBooleanSafe(false);
        return result;
    }

    TMaybe<bool>
    TValidator::CallIsKnown(const TStringBuf vcs, const TStringBuf secret) {
        TStringBuilder headers;

        if (!authToken.empty()) {
            headers << "X-Internal-Token: " << authToken << "\r\n";
        }

        TStringStream body;
        NJson::TJsonWriter json(&body, false);
        json.OpenMap();
        json.Write("vcs", vcs);
        json.Write("secret", secret);
        json.CloseMap();
        json.Flush();

        auto req = NNeh::TMessage(baseUrl + "known", "");
        bool ok = NNeh::NHttp::MakeFullRequest(
            req, headers,
            body.Str(), "application/json",
            NNeh::NHttp::ERequestType::Post
        );
        if (Y_UNLIKELY(!ok)) {
            NSecurityHelpers::LogErr("Failed to make known checking service request");
            return {};
        }

        req.Addr = addr;
        NNeh::TResponseRef resp;
        if (!NNeh::Request(req)->Wait(resp, kHttpDeadline)) {
            NSecurityHelpers::LogErr("Failed to call known checking service", "err", "timed out");
            return {};
        }

        if (Y_UNLIKELY(!resp)) {
            NSecurityHelpers::LogErr("Failed to call known checking service", "err", "empty response");
            return {};
        }

        if (resp->IsError()) {
            NSecurityHelpers::LogErr("Failed to call known checking service", "err", resp->GetErrorText());
            return {};
        }

        NJson::TJsonValue data;
        if (!ReadJsonTree(resp->Data, &data)) {
            NSecurityHelpers::LogErr("Failed to parse known checking service response");
            return {};
        }

        if (!data["ok"].GetBooleanSafe(false)) {
            return {};
        }

        return data["known"].GetBooleanSafe(false);
    }

    TValidatorCache::TValidatorCache(TValidator& validator, size_t cacheSize)
        : validator(validator)
        , cache(*this, cacheSize)
    {
    }

    TMaybe<TResult>
    TValidatorCache::Get(const TStringBuf type, const TStringBuf token) {
        auto result = cache.Get(type, token);
        if (result) {
            return *result;
        }
        return Nothing();
    }

    TResult* TValidatorCache::CreateObject(const TStringBuf type, const TStringBuf token) const {
        return validator.CallBypassCache(type, token);
    }

    TString TValidatorCache::GetKey(const TStringBuf type, const TStringBuf token) const {
        return TString::Join(type, token);
    }

}
