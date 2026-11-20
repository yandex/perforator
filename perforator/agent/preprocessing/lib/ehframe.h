#pragma once

#include "unwind_table_builder.h"

#include <llvm/DebugInfo/DWARF/LowLevel/DWARFUnwindTable.h>

namespace NPerforator::NBinaryProcessing::NUnwind {

    UnwindTable BuildUnwindTableFromEhFrame(llvm::object::ObjectFile* objectFile, const BinaryAnalysisOptions& opts);

    // Converts a single CFI register location into an unwind rule. Exposed for tests.
    void FillRegisterLocation(UnwindRule* rule, const llvm::dwarf::UnwindLocation& loc);

} // namespace NPerforator::NBinaryProcessing::NUnwind
