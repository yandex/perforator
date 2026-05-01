#include "searcher.h"
#include "masker.h"

#include <security/ant-secret/snooper/internal/searchers/all.h>

namespace NSnooperInt {
    TSearcher::TSearcher(TSearchersStore* seachers)
        : seachers(seachers)
    {}

    NSecret::TSecretList TSearcher::Search(TStringBuf data, bool validOnly) const {
        return seachers->Search(data, validOnly);
    }

    NSecret::TSecretList TSearcher::Mask(TString& data, bool validOnly) const {
        auto secrets = this->Search(data, validOnly);
        if (!secrets) {
            return {};
        }

        NMask::MaskSecrets(data, secrets);
        return secrets;
    }

    void TSearcher::MaskAtSecrets(TString& data, const NSecret::TSecretList& secrets) const {
        NMask::MaskSecrets(data, secrets);
    }

    void TSearcher::MaskAtSecrets(TArrayRef<char> data, const NSecret::TSecretList& secrets) const {
        NMask::MaskSecrets(data, secrets);
    }
}
