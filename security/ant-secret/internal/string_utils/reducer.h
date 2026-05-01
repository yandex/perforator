#pragma once

#include <util/generic/string.h>
#include <util/generic/strbuf.h>

namespace NStringUtils {
    TString Reduce(const TStringBuf data, size_t important_from, size_t important_len);

}
