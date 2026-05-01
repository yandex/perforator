#pragma once

#include <util/generic/string.h>
#include <util/generic/strbuf.h>

namespace NStringUtils {
    TString Sha1(const char* data, size_t len);

    TString Sha1(const TStringBuf data);

    TString Sha1(const TString& data);

}
