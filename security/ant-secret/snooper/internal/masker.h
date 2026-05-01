#pragma once

#include <util/generic/array_ref.h>
#include <util/generic/strbuf.h>

#include <security/ant-secret/snooper/internal/secret/secret.h>

namespace NSnooperInt::NMask {
    void MaskSecrets(TString& data, const NSecret::TSecretList& secrets);

    void MaskSecrets(TArrayRef<char> data, const NSecret::TSecretList& secrets);
}
