#include "proto_searcher.h"
#include "masker.h"

#include <security/ant-secret/snooper/internal/searchers/all.h>
#include <security/ant-secret/internal/protobuf/protobuf.h>


namespace NSnooperInt {
    TProtoSearcher::TProtoSearcher(TSearchersStore* seachers)
        : seachers(seachers)
    {}

    NSecret::TSecretList TProtoSearcher::Search(TStringBuf data, bool validOnly) const {
        NSecret::TSecretList secrets;
        auto consumer = [this, validOnly, &secrets](size_t offset, TStringBuf value) -> void {
            for (auto& secret : seachers->Search(value, validOnly)) {
                secret.SecretPos.From += offset;
                secret.MaskPos.From += offset;
                secrets.push_back(secret);
            }
        };

        NAnt::NProtobuf::TVisitor(consumer).VisitWire(data);
        return secrets;
    }

    NSecret::TSecretList TProtoSearcher::Mask(TString& data, bool validOnly) const {
        auto secrets = this->Search(data, validOnly);
        if (!secrets) {
            return {};
        }

        NMask::MaskSecrets(data, secrets);
        return secrets;
    }

    void TProtoSearcher::MaskAtSecrets(TString& data, const NSecret::TSecretList& secrets) const {
        NMask::MaskSecrets(data, secrets);
    }

    void TProtoSearcher::MaskAtSecrets(TArrayRef<char> data, const NSecret::TSecretList& secrets) const {
        NMask::MaskSecrets(data, secrets);
    }
}
