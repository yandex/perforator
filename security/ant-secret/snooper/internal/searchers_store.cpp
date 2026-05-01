#include "searchers_store.h"

#include <security/ant-secret/internal/re2utils/options.h>
#include <security/ant-secret/snooper/internal/searchers/all.h>

#include <util/generic/strbuf.h>
#include <util/generic/set.h>

namespace NSnooperInt {
    TSearchersStore::TSearchersStore(NSnooperInt::TContext ctx, NSecret::TSecretTypes neededSecrets)
        : preMatcher(NRe2Utils::DefaultSetOptions(), re2::RE2::UNANCHORED)
    {
        if (neededSecrets & NSecret::ESecretType::YSession) {
            searchers.push_back(MakeHolder<NSearchers::TYandexSession>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YOAuth) {
            searchers.push_back(MakeHolder<NSearchers::TYandexOAuth>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::TVMTicket) {
            searchers.push_back(MakeHolder<NSearchers::TTvmServiceTicket>(ctx));
            searchers.push_back(MakeHolder<NSearchers::TTvmUserTicket>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::TVMSecret) {
            searchers.push_back(MakeHolder<NSearchers::TTvmSecret>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::S3Presign) {
            searchers.push_back(MakeHolder<NSearchers::TS3PresignV2>(ctx));
            searchers.push_back(MakeHolder<NSearchers::TS3PresignV4>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::S3SecretKey) {
            searchers.push_back(MakeHolder<NSearchers::TS3SecretKey>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::MdsSign) {
            searchers.push_back(MakeHolder<NSearchers::TMdsSign>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YCApiKey) {
            searchers.push_back(MakeHolder<NSearchers::TYCApiKey>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YCCookie) {
            searchers.push_back(MakeHolder<NSearchers::TYCCookie>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YCToken) {
            searchers.push_back(MakeHolder<NSearchers::TYCToken>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YCRefreshToken) {
            searchers.push_back(MakeHolder<NSearchers::TYCRefreshToken>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YCStaticCred) {
            searchers.push_back(MakeHolder<NSearchers::TYCStaticCred>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YCOAuthSecret) {
            searchers.push_back(MakeHolder<NSearchers::TYCOAuthSecret>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::JwtToken) {
            searchers.push_back(MakeHolder<NSearchers::TJwt>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::Gitlab) {
            searchers.push_back(MakeHolder<NSearchers::TGitlab>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::UrlPassword) {
            searchers.push_back(MakeHolder<NSearchers::TUrlPassword>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YRoboPassword) {
            searchers.push_back(MakeHolder<NSearchers::TYRoboPassword>(ctx));
            searchers.push_back(MakeHolder<NSearchers::TYRoboPasswordV2>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YHomoPassword) {
            searchers.push_back(MakeHolder<NSearchers::TYHomoPassword>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::MarketPull) {
            searchers.push_back(MakeHolder<NSearchers::TMatketPull>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::MsCookie) {
            searchers.push_back(MakeHolder<NSearchers::TMsCookie>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::DevApiKey) {
            searchers.push_back(MakeHolder<NSearchers::TDevApiKey>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::BoxberryToken) {
            searchers.push_back(MakeHolder<NSearchers::TBoxberry>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::GeoSharingToken) {
            searchers.push_back(MakeHolder<NSearchers::TGeoSharing>(ctx));
        }

        if (neededSecrets & NSecret::ESecretType::YFToken) {
            searchers.push_back(MakeHolder<NSearchers::TYFToken>(ctx));
        }

        {
            size_t keyID = 0;
            for (size_t searcherId = 0; searcherId < searchers.size(); ++searcherId) {
                if (!searchers[searcherId]->Compile()) {
                    ythrow TSystemError() << "searcher " << searchers[searcherId]->Name() << " compilation failed";
                }

                const auto& patterns = searchers[searcherId]->SearchPatterns();
                for (size_t patternId = 0; patternId < patterns.size(); ++patternId) {
                    int res = preMatcher.Add(re2::StringPiece(patterns[patternId]), nullptr);
                    Y_ASSERT(res == (int)keyID);

                    searchersInfo.emplace_back(searcherId, patternId);
                    keyID++;
                }
            }

            if (!preMatcher.Compile()) {
                ythrow TSystemError() << "re2 compilation failed";
            }
        }
    }

    NSecret::TSecretList TSearchersStore::Search(TStringBuf data, bool validOnly) const {
        TVector<int> matchedSearchers;
        if (!preMatcher.Match(re2::StringPiece(data.data(), data.size()), &matchedSearchers)) {
            return {};
        }

        ::Sort(matchedSearchers);
        NSecret::TSecretList secrets;
        size_t lastID = -1;
        for (size_t curID : matchedSearchers) {
            if (lastID == curID) {
                continue;
            }

            if (curID >= searchersInfo.size()) {
                Y_ASSERT(curID < searchersInfo.size());
                continue;
            }

            lastID = curID;
            const TSearcherInfo searcher = searchersInfo[curID];
            TMaybe<NSecret::TSecret> secret;
            NSearchers::TSearchRequest req {
                .data = data,
                .keyId = searcher.patternID,
            };

            if (validOnly) {
                searchers[searcher.searcherID]->SearchValidatedInto(req, secrets);
            } else {
                searchers[searcher.searcherID]->SearchInto(req, secrets);
            }
        }

        return secrets;
    }
}
