#pragma once

#ifdef __GNUC__
#pragma GCC diagnostic push
#pragma GCC diagnostic ignored "-Wunused-parameter"
#endif

#ifndef LLVM_SUPPORT_LOCALE_H
#define LLVM_SUPPORT_LOCALE_H

#include "llvm/Support/Compiler.h"

namespace llvm {
class StringRef;

namespace sys {
namespace locale {

LLVM_ABI int columnWidth(StringRef s);
LLVM_ABI bool isPrint(int c);
}
}
}

#endif // LLVM_SUPPORT_LOCALE_H

#ifdef __GNUC__
#pragma GCC diagnostic pop
#endif
