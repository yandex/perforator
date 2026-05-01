#include "whole.h"

#include <security/ant-secret/internal/re2utils/options.h>

#include <util/string/vector.h>
#include <util/string/join.h>
#include <util/string/subst.h>
#include <util/string/builder.h>

namespace NSearchers {
    namespace  {
        TString urlifyPattern(const TString& pattern) {
            auto out = SubstGlobalCopy(pattern, "\\:", "(:|%3[Aa])");
            SubstGlobal(out, "\\|", "(\\||%7[Cc])");
            SubstGlobal(out, "\\.", "(\\.|%2[Ee])");
            return out;
        }

        bool isReChar(char ch) {
            return ".^$*+-?()[]{}\\|-/"sv.find(ch) != std::string_view::npos;
        }

        TVector<TString> genReEscaped(const TStringBuf in) {
            TVector<TString> out;
            out.reserve(in.size());

            TString seq;
            seq.reserve(2);
            for (auto ch : in) {
                seq.clear();
                if (isReChar(ch)) {
                    seq.append('\\');
                }
                seq.append(ch);
                out.emplace_back(seq);
            }

            return out;
        }
    }

    namespace {
        struct TPatternBound {
            TString left;
            TString right;
        };

        const TVector<TPatternBound> genPatternBounds() {
            constexpr TStringBuf leftBounds = "!#$&()*%,.\\:;<=>?@[]^{|}~'`\" \f\n\r\t\v\x1C\x1D\x1E\x1F";
            constexpr TStringBuf rightBounds = "!#$&()*%,.\\:;<>?@[]^{|}~'`\" \f\n\r\t\v\x1C\x1D\x1E\x1F";

            return {
                {
                    .left="(?:" + JoinSeq("|", genReEscaped(leftBounds)) + "|%[0-9a-fA-F]{2}|^)",
                    .right="(?:" + JoinSeq("|", genReEscaped(rightBounds)) + "|$)",
                },
            };
        }
    }

    bool TWhole::Compile() {
        const TVector<TString> patterns = Patterns();
        Y_ASSERT(patterns.size());
        bool urlify = Uglified();

        const TVector<TPatternBound> bounds = genPatternBounds();
        Y_ASSERT(bounds.size());

        for (size_t patternID = 0; patternID < patterns.size(); ++patternID) {
            auto pattern = patterns[patternID];
            if (urlify) {
                pattern = urlifyPattern(pattern);
            }

            for (auto&& bound : bounds) {
                TString rawPattern = bound.left + "(" + pattern + ")" + bound.right;
                patternsInfo.push_back(TPattern{
                    .patternID = patternID,
                    .re = MakeHolder<re2::RE2>(rawPattern, NRe2Utils::DefaultReOptions()),
                });

                if (!patternsInfo.back().re->ok()) {
                    return false;
                };
            }
        }

        return true;
    }

    TVector<TString> TWhole::SearchPatterns() const {
        TVector<TString> out;
        out.reserve(patternsInfo.size());
        for (auto&& pi : patternsInfo) {
            auto pattern = pi.re->pattern();
            out.push_back(pattern.data());
        }
        return out;
    }

    bool TWhole::SearchInto(const TSearchRequest &req, NSecret::TSecretList& out) const {
        if (req.keyId >= patternsInfo.size()) {
            return false;
        }

        const auto& patternInfo = patternsInfo[req.keyId];
        re2::StringPiece in(req.data);

        bool haveSomething = false;
        re2::StringPiece secretView;
        while (RE2::FindAndConsume(&in, *patternInfo.re, &secretView)) {
            TStringBuf rawSecret{secretView.data(), secretView.size()};
            TString secret = decodeSecret(rawSecret);
            if (!IsSecret(patternInfo.patternID, secret)) {
                continue;
            }

            haveSomething = true;
            size_t startPos = secretView.data() - req.data.data();
            size_t len = secretView.size();
            auto mask = MaskSecret(patternInfo.patternID, rawSecret);
            // add initial position to the mask info
            mask.From += startPos;
            out.push_back(NSecret::TSecret{
                .Type = SecretType(),
                .Secret = secret,
                .SecretPos = NSecret::TPos{
                    .From = startPos,
                    .Len = len,
                },
                .MaskPos = mask,
            });
        }

        return haveSomething;
    }

    bool TWhole::SearchValidatedInto(const TSearchRequest &req, NSecret::TSecretList& out) const {
        if (!ctx.Validator) {
            return false;
        }

        NSecret::TSecretList secrets;
        bool haveSomething = SearchInto(req, secrets);
        if (!haveSomething) {
            return false;
        }

        TMaybe<bool> forced = ForceValid();
        if (forced.Defined()) {
            if (!forced.GetRef()) {
                return false;
            }

            out.insert(out.end(), secrets.begin(), secrets.end());
            return true;
        }

        haveSomething = false;
        for (auto&& secret : secrets) {
            const auto &info = ctx.Validator->Call(Name(), secret.Secret);
            if (!info || !info->Valid) {
                continue;
            }

            haveSomething = true;
            out.push_back(secret);
        }

        return haveSomething;
    }

    bool TWhole::Uglified() const {
        return true;
    }
}
