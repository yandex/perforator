#pragma once

#include "secret_types.h"

#include <util/generic/fwd.h>
#include <util/generic/string.h>
#include <util/generic/vector.h>


namespace NSecret {
    struct TPos {
        size_t From;
        size_t Len;
    };

    struct TSecret {
        ESecretType Type = ESecretType::Unknown;
        TString Secret;
        TPos SecretPos;
        TPos MaskPos;
    };

    using TSecretList = TVector<TSecret>;
}
