#pragma once

#include <security/ant-secret/internal/validation/validator.h>

#include <util/generic/ptr.h>

namespace NSnooperInt {
    struct TContext {
        TAtomicSharedPtr<NValidation::IValidator> Validator = nullptr;

        TContext() {
            Validator = MakeAtomicShared<NValidation::TValidator>();
        }
    };
}
