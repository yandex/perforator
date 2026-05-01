#pragma once

#include <util/generic/array_ref.h>

#include <security/ant-secret/snooper/internal/searchers_store.h>
#include <security/ant-secret/snooper/internal/secret/secret.h>

namespace NSnooperInt {
    class TSearcher {
    public:
        explicit TSearcher(TSearchersStore* seachers);

        NSecret::TSecretList Search(TStringBuf data, bool validOnly = false) const;

        NSecret::TSecretList Mask(TString& data, bool validOnly = false) const;

        void MaskAtSecrets(TString& data, const NSecret::TSecretList& secrets) const;

        void MaskAtSecrets(TArrayRef<char> data, const NSecret::TSecretList& secrets) const;

    protected:
        TSearchersStore* seachers;
    };
}
