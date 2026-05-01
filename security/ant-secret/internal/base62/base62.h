#pragma once

#include <util/generic/strbuf.h>

namespace NBase62 {

    template <typename T>
    T DecodeNum(TStringBuf in);

    bool IsToken(TStringBuf in, size_t prefixLen = 4, size_t crcLen = 6);
}
