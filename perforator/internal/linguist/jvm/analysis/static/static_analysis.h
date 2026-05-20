#pragma once

#include <string_view>
#include <cstddef>

#include <perforator/internal/linguist/jvm/analysis/api/api.h>

namespace NPerforator::NLinguist::NJvm {


// Information extracted from the VMStructs class.
// See https://github.com/openjdk/jdk/blob/89f9268ed7c2cb86891f23a10482cd459454bd32/src/hotspot/share/runtime/vmStructs.hpp#L34
struct TVMStructsAddresses {
    constexpr static std::string_view StructsAddressSym = "gHotSpotVMStructs";
    const void* StructsAddress;
    constexpr static std::string_view TypesAddressSym = "gHotSpotVMTypes";
    const void* TypesAddress;
};

TJvmAnalysis ProcessDynamicLinkedJVM(TVMStructsAddresses);

TJvmAnalysis ProcessJVMHeaders();


} // namespace NPerforator::NLinguist::NJvm
