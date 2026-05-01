#pragma once

#include <util/generic/strbuf.h>

namespace NJwt {
    bool IsToken(TStringBuf token);

    bool IsHeader(TStringBuf rawHeader);

    bool IsPayload(TStringBuf rawHeader);
}
