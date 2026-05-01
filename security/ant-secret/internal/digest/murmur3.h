#pragma once

#include <util/generic/strbuf.h>

namespace NDigest {
    uint32_t Murmur3_32(TStringBuf in, uint32_t seed = 0);
    uint32_t Murmur3_32(const void* key, size_t len, ui32 seed);
}
