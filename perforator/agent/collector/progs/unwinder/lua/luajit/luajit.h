#pragma once

// Macro used to mark parts of the code that are disabled for the perforator in
// order to compile in BPF.
#define DISABLED_BY_PERFORATOR

#include <bpf/bpf.h>

#include "lj_arch.h"
#include "lj_bc.h"
#include "lj_debug.h"
#include "lj_def.h"
#include "lj_dispatch.h"
#include "lj_frame.h"
#include "lj_ir.h"
#include "lj_jit.h"
#include "lj_obj.h"
#include "lj_state.h"
#include "lj_stdint.h"
#include "lua.h"
#include "luaconf.h"

// TODO: Temp until we strip headers
_Static_assert(offsetof(lua_State, base) == 32, "");
_Static_assert(offsetof(lua_State, top) == 40, "");
_Static_assert(offsetof(lua_State, maxstack) == 48, "");
_Static_assert(offsetof(lua_State, stack) == 56, "");
_Static_assert(offsetof(lua_State, cframe) == 80, "");
_Static_assert(offsetof(lua_State, stacksize) == 88, "");
_Static_assert(offsetof(global_State, vmstate) == 184, "");
