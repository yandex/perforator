#include "reducer.h"

#include <util/string/builder.h>

namespace NStringUtils {
    namespace {
        constexpr size_t kContextBytes = 256;
        constexpr TStringBuf kRedocePlaceholder = "...[truncated]...";
    }

    TString Reduce(const TStringBuf data, size_t important_from, size_t important_len) {
        if (data.size() <= (kContextBytes*6 + important_len)) {
            return TString(data);
        }

        if (important_from + important_len >= data.size()) {
            return TString(data);
        }

        TStringBuilder out;
        out.reserve(kContextBytes*6);
        if (important_from > kContextBytes*2) {
            out
                << data.substr(0, kContextBytes)
                << kRedocePlaceholder
                << data.substr(important_from-kContextBytes, kContextBytes);
        } else {
            out << data.substr(0, important_from);
        }

        out << data.substr(important_from, important_len);

        if (important_from + important_len < data.size() - kContextBytes*2) {
            out
                << data.substr(important_from + important_len, kContextBytes)
                << kRedocePlaceholder
                << data.substr(data.size() - kContextBytes, kContextBytes);
        } else {
            out << data.substr(important_from + important_len, data.size() - important_from - important_len);
        }

        return out;
    }

}
