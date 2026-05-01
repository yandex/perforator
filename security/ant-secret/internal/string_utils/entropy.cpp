#include "entropy.h"

#include <util/generic/vector.h>
#include <util/generic/string.h>

#include <cmath>

namespace NStringUtils {
    float ShannonEntropy(const TStringBuf in, const char table[]) {
        static_assert(sizeof(TString::TChar) == sizeof(ui8), "Entropy required int8-based TString::TChar");

        TVector<size_t> counts(256);
        for (auto ch : in) {
            if (!table[static_cast<ui8>(ch)]) {
                return -1.0f;
            }
            ++counts[static_cast<ui8>(ch)];
        }

        float entropy = 0.0f;

        for (auto c : counts) {
            if (c != 0) {
                float p = static_cast<float>(c) / in.size();
                entropy -= p * std::log2(p);
            }
        }

        return entropy;
    }

}
