#include "prefixed.h"

#include <security/ant-secret/internal/re2utils/options.h>

#include <util/string/vector.h>
#include <util/string/join.h>

namespace NSearchers {
    namespace {
        struct TPatternBound {
            TString left;
            TString right;
        };

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

        // TODO(buglloc): merge with TWhole plz
        const TPatternBound genPatternBounds() {
            constexpr TStringBuf leftBounds = "!#$&()*%,.\\:;<=>?@[]^{|}~'`\" \f\n\r\t\v\x1C\x1D\x1E\x1F";
            constexpr TStringBuf rightBounds = "!#$&()*%,.\\:;<>?@[]^{|}~'`\" \f\n\r\t\v\x1C\x1D\x1E\x1F";

            return {
                .left="(?:" + JoinSeq("|", genReEscaped(leftBounds)) + "|%[0-9a-fA-F]{2}|^)",
                .right="(?:" + JoinSeq("|", genReEscaped(rightBounds)) + "|$)"
            };
        }

    }

    bool TPrefixed::Compile() {
        const TVector<TString> rawKeyPatterns = KeyPatterns();
        Y_ASSERT(rawKeyPatterns.size());
        const TVector<TString> rawValuePatterns = ValuePatterns();
        Y_ASSERT(rawValuePatterns.size());
        auto quotedSeparators = QuotedSeparator();
        Y_ASSERT(quotedSeparators.size());
        auto kvSeparators = KvSeparators();
        Y_ASSERT(kvSeparators.size());
        auto patternBounds = genPatternBounds();
        Y_ASSERT(patternBounds.left.size() && patternBounds.right.size());

        TVector<TString> separators;
        separators.insert(separators.end(), kvSeparators.begin(), kvSeparators.end());
        separators.insert(separators.end(), quotedSeparators.begin(), quotedSeparators.end());

        TString kvSeparator = "(?:" + JoinSeq("|", separators) + ")";
        for (auto&& keyPattern : rawKeyPatterns) {
            keyPatterns.push_back(patternBounds.left + "(?i:" + keyPattern + ")" + kvSeparator);
            size_t patternID = 0;
            for (auto&& valuePattern : rawValuePatterns) {
                TString rawPattern = patternBounds.left + "(?i:" + keyPattern + ")" + kvSeparator + "(" + valuePattern + ")" + patternBounds.right;
                valuePatterns.push_back(TPattern{
                    .patternID = patternID,
                    .re = MakeHolder<re2::RE2>(rawPattern, NRe2Utils::DefaultReOptions()),
                });

                if (!valuePatterns.back().re->ok()) {
                    return false;
                }

                patternID++;
            }
        }

        return true;
    }

    TVector<TString> TPrefixed::SearchPatterns() const {
        return keyPatterns;
    }

    bool TPrefixed::SearchInto(const TSearchRequest& req, NSecret::TSecretList& out) const {
        bool haveSomething = false;

        for (auto&& patternInfo : valuePatterns) {
            re2::StringPiece in(req.data);
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
        }

        return haveSomething;
    }

    bool TPrefixed::SearchValidatedInto(const TSearchRequest& req, NSecret::TSecretList& out) const {
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

    TVector<TString> TPrefixed::KvSeparators() const {
        return {
            R"( )",
            R"(\s*: )",
            R"(\\?=)",
        };
    }

    TVector<TString> TPrefixed::QuotedSeparator() const {
        const TVector<TString> quotes = {
            R"(\\*")",
            R"(\\*%22)",
            R"(\\*')",
        };

        const TVector<TString> separators = {
            R"(\s*:\s*)",
            R"(\s*%3[Aa]\s*)",
            R"((?:\s|\\)*=(?:\s|\\)*)",
            R"((?:\s|\\)*=>(?:\s|\\)*)",
        };

        TVector<TString> out;
        for (auto&& quote : quotes) {
            for (auto&& sep : separators) {
                out.push_back(quote+sep+quote);
            }
        }
        return out;
    }
}
