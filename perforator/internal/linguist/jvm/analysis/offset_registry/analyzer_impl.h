#pragma once

#include "offset_registry.h"

#include <perforator/internal/linguist/jvm/analysis/api/api.h>

namespace NPerforator::NLinguist::NJvm {

struct TOffsetRegistryAnalysisOptions {
    bool IncludeAddresses = false;
};

uint64_t FindMajorVersionAddress(const TJvmMetadata& metadata);
uint64_t FindMinorVersionAddress(const TJvmMetadata& metadata);
uint64_t FindSecurityVersionAddress(const TJvmMetadata& metadata);

TJvmAnalysis ProcessOffsetRegistry(const TJvmMetadata& metadata, TOffsetRegistryAnalysisOptions options, ui32 version);

}
