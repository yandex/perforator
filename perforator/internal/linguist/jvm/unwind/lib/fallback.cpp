#include "cheatsheet.h"
#include "offsets.h"

#include <util/generic/yexception.h>


namespace NPerforator::NLinguist::NJvm {

namespace {
[[noreturn]]
void Unimplemented() {
    throw yexception() << "no JDK headers configured, see perforator/internal/linguist/jvm/unwind/README.md";
}
}


TOffsets TOffsets::Get() {
    Unimplemented();
}

TJvmVersionInfo GetFromVMStructs(TVMStructsAddresses) {
    Unimplemented();
}

} // namespace NPerforator::NLinguist::NJvm
