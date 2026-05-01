#pragma once

#include <util/generic/strbuf.h>

namespace NBlackBox {
    bool IsEmbeddedV2Token(const TStringBuf token);

    bool IsEmbeddedV3Token(const TStringBuf token);
}
