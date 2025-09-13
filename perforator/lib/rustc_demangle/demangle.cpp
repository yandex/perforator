#include "demangle.h"

#include <util/charset/wide.h>
#include <util/generic/algorithm.h>
#include <util/generic/strbuf.h>
#include <util/generic/string.h>
#include <util/string/ascii.h>
#include <util/string/cast.h>


namespace NPerforator::NDemangle {

// See https://github.com/rust-lang/rustc-demangle/blob/main/src/legacy.rs
static constexpr size_t rustcHashSuffixLength = 16 + 1 + 2; // ::h + 16 hex digits

bool LooksLikeRustLegacySymbol(TStringBuf symbol) {
    TStringBuf hash = symbol.Last(rustcHashSuffixLength);
    if (hash.size() != rustcHashSuffixLength) {
        return false;
    }

    if (!hash.SkipPrefix("::h")) {
        return false;
    }

    return AllOf(hash, IsAsciiHex<char>);
}

// See https://github.com/rust-lang/rustc-demangle/blob/83f1bbd6793a2dbd5fa94b185a0cd9bb98d8332f/src/legacy.rs#L144-L153
TMaybe<char32_t> UnescapeDollaredChar(TStringBuf str) {
    if (str.SkipPrefix("u")) {
        ui32 result{};
        if (TryIntFromString<16>(str, result)) {
            return result;
        }
        return Nothing();
    }

    static constexpr std::pair<std::string_view, char> replacements[] = {
        {"SP", '@'},
        {"BP", '*'},
        {"RF", '&'},
        {"LT", '<'},
        {"GT", '>'},
        {"LP", '('},
        {"RP", ')'},
        {"C", ','},
    };
    for (auto [from, to] : replacements) {
        if (str == from) {
            return static_cast<char32_t>(to);
        }
    }

    return Nothing();
}

struct TUndollaredCodepoint {
    TStringBuf Escaped;
    char32_t Unescaped;
};

TMaybe<TUndollaredCodepoint> ChopDollaredChar(TStringBuf symbol) {
    if (!symbol.SkipPrefix("$")) {
        return Nothing();
    }

    size_t end = symbol.find('$');
    if (end == TStringBuf::npos) {
        return Nothing();
    }

    auto escaped = TStringBuf{symbol}.Head(end);
    TMaybe<char32_t> unescaped = UnescapeDollaredChar(escaped);
    if (!unescaped) {
        return Nothing();
    }

    return TUndollaredCodepoint{
        .Escaped = escaped,
        .Unescaped = *unescaped,
    };
}

std::string MaybeDemangleRustcName(std::string&& func) {
    if (!LooksLikeRustLegacySymbol(func)) {
        return std::move(func);
    }

    size_t pos = 0;

    for (size_t i = 0; i + rustcHashSuffixLength < func.size(); ++i) {
        char c = func[i];
        if (c == '$') {
            TMaybe<TUndollaredCodepoint> codepoint = ChopDollaredChar(TStringBuf{func}.substr(i));
            if (codepoint) {
                TString unescaped = WideToUTF8(TUtf32StringBuf{&codepoint->Unescaped, 1});
                Y_ENSURE(codepoint->Escaped.size() >= unescaped.size());
                func.replace(pos, unescaped.size(), unescaped);
                pos += unescaped.size();
                i += 1 + codepoint->Escaped.size();
            } else {
                func[pos++] = c;
            }
        } else {
            func[pos++] = c;
        }
    }
    func.resize(pos);

    return std::move(func);
}

} // namespace NPerforator::NDemangle
