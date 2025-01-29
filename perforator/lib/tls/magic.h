#pragma once

#include "magic_bytes.h"

#include <util/system/types.h>

#include <algorithm>
#include <array>


namespace NPerforator::NThreadLocal {

constexpr inline const ui8 kMagic[7] = {
    PERFORATOR_TLS_MAGIC_BYTE_0,
    PERFORATOR_TLS_MAGIC_BYTE_1,
    PERFORATOR_TLS_MAGIC_BYTE_2,
    PERFORATOR_TLS_MAGIC_BYTE_3,
    PERFORATOR_TLS_MAGIC_BYTE_4,
    PERFORATOR_TLS_MAGIC_BYTE_5,
    PERFORATOR_TLS_MAGIC_BYTE_6
};

enum class EVariableKind : ui8 {
    Invalid = 0,
    UnsignedInt64 = 1,
    StringPointer = 2,

};

////////////////////////////////////////////////////////////////////////////////

struct TMagic {
    ui8 Magic[7];
    ui8 Kind;
};

static_assert(sizeof(TMagic) == 8);

////////////////////////////////////////////////////////////////////////////////

inline TMagic MakeMagic(EVariableKind kind) {
    TMagic magic{};
    std::copy(std::begin(kMagic), std::end(kMagic), std::begin(magic.Magic));
    magic.Kind = static_cast<ui8>(kind);
    return magic;
}

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforato::NThreadLocal
