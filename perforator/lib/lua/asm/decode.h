#pragma once

#if defined(__aarch64__)

#include "arm/decode.h"

#elif defined(__x86_64__)

#include "x86/decode.h"

#else

#error "Unsupported architecture"

#endif

namespace NPerforator::NLinguist::NLua::NAsm {

#if defined(__aarch64__)

using NArm::DecodeLuaClose;
using NArm::DecodeLuaOpenJit;
using NArm::DecodeLjDispatchUpdate;

#elif defined(__x86_64__)

using NX86::DecodeLuaClose;
using NX86::DecodeLuaOpenJit;
using NX86::DecodeLjDispatchUpdate;

#endif

}
