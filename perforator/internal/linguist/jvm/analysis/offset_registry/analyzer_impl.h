#pragma once

#include "offset_registry.h"

#include <perforator/internal/linguist/jvm/analysis/api/api.h>

namespace NPerforator::NLinguist::NJvm {

struct TOffsetRegistryAnalysisOptions {
    bool IncludeAddresses = false;
};

TJvmAnalysis ProcessOffsetRegistry(const TJvmMetadata& metadata, TOffsetRegistryAnalysisOptions options, ui32 version);

}
