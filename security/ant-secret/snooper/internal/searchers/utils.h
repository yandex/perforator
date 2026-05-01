#pragma once

#include <util/string/util.h>

#include <library/cpp/string_utils/quote/quote.h>

namespace NSearchers {
    inline TString unslash(const TStringBuf in) {
        if (in.empty()) {
            return TString();
        }

        if (!in.Contains('\\')) {
            return TString{in};
        }

        TString out;
        out.reserve(in.length());

        for (size_t i = 0; i < in.length(); ++i) {
            char c = in[i];
            if (c == '\\' && (i + 1 >= in.length() || !std::isalnum(in[i + 1]))) {
                // skip backslash only if it's the last character or followed by non-alphanumeric
                continue;
            }

            out.push_back(in[i]);
        }

        return out;
    }

    inline TString decodeSecret(const TStringBuf data) {
        TString out = unslash(data);

        // decode urlencoded
        if (out.Contains("%")) {
            UrlUnescape(out);
        }

        return out;
    }
}
