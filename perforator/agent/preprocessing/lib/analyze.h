#pragma once

#include <perforator/agent/preprocessing/proto/parse/parse.pb.h>

#include <util/stream/input.h>
#include <util/stream/output.h>


namespace NPerforator::NBinaryProcessing {

////////////////////////////////////////////////////////////////////////////////

BinaryAnalysis AnalyzeBinary(const char* path);

void SerializeBinaryAnalysis(BinaryAnalysis&& analysis, IOutputStream* out);

BinaryAnalysis DeserializeBinaryAnalysis(IInputStream* input);

////////////////////////////////////////////////////////////////////////////////

} // namespace NPerforator::NBinaryProcessing
