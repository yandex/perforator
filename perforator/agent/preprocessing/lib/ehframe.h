#pragma once

#include <perforator/agent/preprocessing/proto/unwind/table.pb.h>

#include <llvm/Object/ObjectFile.h>

namespace NPerforator::NBinaryProcessing::NUnwind {

void DifferentiateUnwindTable(NUnwind::UnwindTable& table);
void IntegrateUnwindTable(NUnwind::UnwindTable& table);

void DeltaEncode(NUnwind::UnwindTable& table);

NPerforator::NBinaryProcessing::NUnwind::UnwindTable BuildUnwindTable(llvm::object::ObjectFile* objectFile);

} // namespace NPerforator::NBinaryProcessing::NUnwind