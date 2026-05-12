package hardcode

var (
	// Lua -> C
	LuaInternalEvaluationFunctions = map[string]bool{
		//"[builtin] ffi.meta.__call#171": true,
	}

	// These functions serve as an entrypoint to lua evaluation loop
	// C -> Lua
	LuaAPIEvaluationFunctions = map[string]bool{

		// FIXME : clarify
		"lua_call":   true,
		"lua_pcall":  true,
		"lua_cpcall": true,
		"lua_yeild":  true,
		"lua_resume": true,
	}
)
