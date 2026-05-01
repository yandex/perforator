#pragma once

#include "prefixed.h"

namespace NSearchers {
    class TMdsSign: public TPrefixed {
    public:
        using TPrefixed::TPrefixed;

        TString Name() const override;

    protected:
        TVector<TString> KeyPatterns() const override;

        TVector<TString> ValuePatterns() const override;

        bool IsSecret(size_t id, TStringBuf secret) const override;

        NSecret::TPos MaskSecret(size_t id, const TStringBuf rawSecret) const override;

        NSecret::ESecretType SecretType() const override;

        TMaybe<bool> ForceValid() const override;
    };

}
