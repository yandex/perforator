#pragma once

#include <util/generic/strbuf.h>

namespace NStringUtils {
    float ShannonEntropy(const TStringBuf data, const char table[]);

}
