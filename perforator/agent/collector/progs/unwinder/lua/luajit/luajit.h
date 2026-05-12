#pragma once

// Macro used to mark parts of the code that are disabled for the perforator in
// order to compile in BPF.
#define DISABLED_BY_PERFORATOR

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
