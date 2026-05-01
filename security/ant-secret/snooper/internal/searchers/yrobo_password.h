#pragma once

#include "whole.h"

namespace NSearchers {
    class TYRoboPassword: public TWhole {
    public:
        using TWhole::TWhole;

        TString Name() const override;

    protected:
        TVector<TString> Patterns() const override;

        bool IsSecret(size_t id, TStringBuf token) const override;

        NSecret::TPos MaskSecret(size_t id, const TStringBuf rawSecret) const override;

        NSecret::ESecretType SecretType() const override;

        TMaybe<bool> ForceValid() const override;
    };

}
