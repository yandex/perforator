#pragma once

#include <perforator/agent/preprocessing/proto/jvm/jvm.pb.h>

#include <util/generic/map.h>

namespace NPerforator::NLinguist::NJvm {

struct TJvmAnalysis {
    NPerforator::NBinaryProcessing::NJvm::Cheatsheet Cheatsheet;

    NPerforator::NBinaryProcessing::NJvm::Version Version;
};

}
