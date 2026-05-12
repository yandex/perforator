local HASH = {}
local ffi = require("ffi")

ffi.cdef[[
double sqrt(double x);

typedef int (*qsort_cmp)(const void *, const void *);

void qsort(void *base, size_t nmemb, size_t size, qsort_cmp compar);
]]

-- just enough iterations to collect some samples
local ITER = 10000

for i = 0, ITER do
    HASH[tostring(i)] = i
end

local function hash_walk(seed)
    local accumulator = seed
    for i = 0, ITER do
        local k = tostring((i * accumulator) % 4096)
	accumulator = accumulator + (HASH[k] or 0)
    end
    return accumulator
end

-- sink to prevent return value optimization
local function sink(x)
    return x + 1
end

-- just a simple Lua -> Lua invocation (FRAME_LUA)

local function plain_lua_call(x)
    return hash_walk(x) + (x % 2)
end

local function test_plain_lua_call()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + plain_lua_call(i)
    end
    return accumulator
end

-- variadic function (FRAME_VARG)

local function variadic(x, ...)
    local accumulator = x
    local number_of_arguments = select("#", ...)
    for i = 1, number_of_arguments do
        -- consume variadic argument
        accumulator = accumulator + select(i, ...)
    end
    return hash_walk(accumulator)
end

local function test_variadic()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + variadic(i, 1, 2, 3)
    end
    return accumulator
end

-- regular protected call aka pcall (FRAME_PCALL)

local function pcall_hash(x)
    local ok, result = pcall(hash_walk, x)
    if not ok then
        return 0
    end
    return result
end

local function test_pcall()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + pcall_hash(i)
    end
    return accumulator
end

-- pcall with active hook (debug.sethook)

local function noop_hook()
    -- do nothing intentionally
end

local function pcallh_hash(x)
    debug.sethook(noop_hook, "l") -- "l" for line hook
    local ok, result = pcall(hash_walk, x)
    debug.sethook() -- disable formely set hook
    if not ok then
        return 0
    end
    return result
end

local function test_pcallh()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + pcallh_hash(i)
    end
    return accumulator
end

-- protected call of C function

local function pcallc_hash(x)
    local ok, result = pcall(ffi.C.sqrt, x + 1.0)
    if not ok then
        return 0
    end
    return hash_walk(x + math.floor(result))
end

local function test_pcallc()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + pcallc_hash(i)
    end
    return accumulator
end

-- xpcall, protected call with error handling

local function error_handler(err)
    -- minimal handler, but must be callable
    return err
end

local function xpcall_hash(x)
    local ok, result = xpcall(hash_walk, error_handler, x)
    if not ok then
        return 0
    end
    return result
end

local function test_xpcall()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + xpcall_hash(i)
    end
    return accumulator
end

-- protected call with an actual error

local function error_hash_walk(x)
    if (x % 2) == 0 then
        error("boom")
    end
    return hash_walk(x)
end

local function pcall_error_hash(x)
    local ok, result = pcall(error_hash_walk, x)
    if not ok then
        return 0
    end
    return result
end

local function test_pcall_error()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + pcall_error_hash(i)
    end
    return accumulator
end

-- global function (G)

function global_lua_call(x)
    return hash_walk(x) + (x % 2)
end

local function test_global()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + global_lua_call(i)
    end
    return accumulator
end

-- tail call

local function tail_call(x)
    return hash_walk(x)
end

-- a driver is needed, because tail-call elimination was prevented at the caller site

local function tail_driver(x)
    return tail_call(x)
end

local function test_tail_call()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + tail_driver(i)
    end
    return accumulator
end

-- table function (dot call)

local tbl = {
    table_function = function(x)
        return hash_walk(x) + (x % 2)
    end
}

local function test_table_call()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + tbl.table_function(i)
    end
    return accumulator
end

-- table method (colon call)

local obj = {}

function obj:table_method(x)
    return hash_walk(x + (self.bias or 0)) + (x % 2)
end

obj.bias = 1

local function test_table_method()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + obj:table_method(i)
    end
    return accumulator
end

-- closure / upvalue

local function make_closure(bias)
    return function(x)
        return hash_walk(x + bias) + (x % 2)
    end
end

local closure_function = make_closure(3)

local function test_closure()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + closure_function(i)
    end
    return accumulator
end

-- indirect function call (via variable)

local indirect = hash_walk

local function test_indirect()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + indirect(i)
    end
    return accumulator
end

-- metatable __call

local callable = {}

setmetatable(callable, {
    __call = function(_, x)
        return hash_walk(x) + (x % 2)
    end
})

local function test_metatable_call()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + callable(i)
    end
    return accumulator
end

-- metatable __index

local mt = {
    __index = function(_, k)
        return function(x)
            return hash_walk(x) + (x % 2)
        end
    end
}

local proxy = setmetatable({}, mt)

local function test_metatable_index()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + proxy.fn(i)
    end
    return accumulator
end


-- metatable __newindex

local ni = setmetatable({}, {
    __newindex = function(_, k, v)
        sink(hash_walk(v))
    end
})

local function test_metatable_newindex()
    for i = 0, ITER do
        ni.x = i
    end
    return 0
end

-- metatable __string

local string_object = setmetatable({}, {
    __tostring = function()
        return tostring(hash_walk(1))
    end
})

local function test_metatable_string()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + #tostring(string_object)
    end
    return accumulator
end

-- metatable __pairs

local pairs_table = setmetatable({a=1,b=2,c=3}, {
    __pairs = function(self)
        local function iter(_, k)
            return next(self, k)
        end
        return iter, self, nil
    end
})

local function test_metatable_pairs()
    local accumulator = 0
    for i = 1, ITER do
        for k, v in pairs(pairs_table) do
            accumulator = accumulator + hash_walk(v)
        end
    end
    return accumulator
end

-- coroutine resume

local function coroutine_body(x)
    return hash_walk(x) + (x % 2)
end

local function test_coroutine()
    local accumulator = 0
    for i = 0, ITER do
        local co = coroutine.create(coroutine_body)
        local ok, result = coroutine.resume(co, i)
        if ok then
            accumulator = accumulator + result
        end
    end
    return accumulator
end

-- environment resolution (ENV)

local env = {
    fn = hash_walk
}

local function env_caller(i)
    return fn(i)
end

setfenv(env_caller, env)

local function test_environment()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + env_caller(i)
    end
    return accumulator
end

-- module call

local M = {}

function M.fn(x)
    return hash_walk(x) + (x % 2)
end

local function test_module_call()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + M.fn(i)
    end
    return accumulator
end

-- FFI callback

local N = 1024
local arr = ffi.new("int[?]", N)

for i = 0, N - 1 do
    arr[i] = (i * 17) % 257
end

local function cmp(a, b)
    local x = ffi.cast("const int*", a)[0]
    local y = ffi.cast("const int*", b)[0]

    -- prevent optimization, force work
    local v = hash_walk(x + y)

    if v % 2 == 0 then
        return x - y
    else
        return y - x
    end
end

local cmp_cb = ffi.cast("qsort_cmp", cmp)

local function test_ffi_callback()
    -- qsort is very computation heavy by itself, so reduce a number of iterations
    for i = 0, ITER / 100 do
        ffi.C.qsort(arr, N, ffi.sizeof("int"), cmp_cb)
    end
    return arr[0]
end

-- lua_pcallk https://www.lua.org/manual/5.3/manual.html#lua_pcallk
-- skip, not implemented by LuaJIT https://github.com/LuaJIT/LuaJIT/issues/48

-- __gc metamethod

local gc_anchor = setmetatable({}, { __mode = "v" })

local function make_gc_object(x)
    local ud = newproxy(true)
    getmetatable(ud).__gc = function()
        sink(hash_walk(x))
    end
    return ud
end

local function test_gc()
    collectgarbage("restart")
    for i = 0, ITER do
        gc_anchor[i] = make_gc_object(i)
    end

    -- remove strong references
    for i = 0, ITER do
        gc_anchor[i] = nil
    end

    -- force GC to run finalizers
    collectgarbage()
    collectgarbage()
    collectgarbage()

    return 0
end

-- debug hook

local hook_counter = 0

local function debug_hook()
    hook_counter = hook_counter + 1
    local r = hash_walk(hook_counter)
    sink(r)
end

local function test_debug_hook()
    hook_counter = 0

    -- call hook every 100 instructions
    debug.sethook(debug_hook, "", 100)

    local accumulator = 0
    -- every 100 instructions is very often, reduce numbers of iterations
    for i = 0, ITER / 100 do
        accumulator = accumulator + hash_walk(i)
    end

    debug.sethook() -- remove hook
    return accumulator
end


-- error in FFI callback

local function cmp_error(a, b)
    local x = ffi.cast("const int*", a)[0]
    local y = ffi.cast("const int*", b)[0]

    -- prevent optimization, force work
    local v = hash_walk(x + y)

    if v % 2 == 0 then
        error("boom")
    else
        return y - x
    end
end

local cmp_error_cb = ffi.cast("qsort_cmp", cmp_error)

local function test_error_in_ffi_callback_body()
    ffi.C.qsort(arr, N, ffi.sizeof("int"), cmp_error_cb)
end

local function test_error_in_ffi_callback()
    for i = 0, ITER do
        -- pcall is required in order to avoid failing an entire script from erron within FFI callback
        pcall(test_error_in_ffi_callback_body)
    end
    return arr[0]
end

-- error in GC

local gc_anchor_error = setmetatable({}, { __mode = "v" })

local function make_gc_object_error(x)
    local ud = newproxy(true)
    getmetatable(ud).__gc = function()
        if x % 2 == 0 then
            error("boom")
        end
        sink(hash_walk(x))
    end
    return ud
end

local function test_error_in_gc()
    collectgarbage("restart")
    for i = 0, ITER do
        gc_anchor_error[i] = make_gc_object_error(i)
    end

    -- remove strong references
    for i = 0, ITER do
        gc_anchor_error[i] = nil
    end

    -- force GC to run finalizers
    collectgarbage()
    collectgarbage()
    collectgarbage()

    return 0
end

-- error in debug hook

local hook_error_counter = 0

local function debug_hook_error()
    hook_error_counter = hook_error_counter + 1
    if hook_error_counter % 2 == 0 then
        error("boom")
    end
    local r = hash_walk(hook_error_counter)
    sink(r)
end

local function test_error_in_debug_hook_body(accumulator, i)
    -- call hook every 100 instructions
    debug.sethook(debug_hook_error, "", 100)

    accumulator = accumulator + hash_walk(i)

    debug.sethook() -- remove hook

    return accumulator
end

local function test_error_in_debug_hook()
    hook_counter = 0

    local accumulator = 0
    for i = 0, ITER do
        -- pcall is required in order to avoid failing an entire script from erron within debug hook
        local ok, result = pcall(test_error_in_debug_hook_body, accumulator, i)
        if not ok then
            accumulator = 0
        else
            accumulator = result
        end
    end

    return accumulator
end

-- error in metatable __index

local mt_error = {
    __index = function(_, k)
        return function(x)
            if x % 2 == 0 then
                error("boom")
            end
            return hash_walk(x) + (x % 2)
        end
    end
}

local proxy_error = setmetatable({}, mt_error)

local function test_error_in_metatable_index_body(accumulator, i)
    return accumulator + proxy_error.fn(i)
end

local function test_error_in_metatable_index()
    local accumulator = 0
    for i = 0, ITER do
        local ok, result = pcall(test_error_in_metatable_index_body, accumulator, i)
        if not ok then
            accumulator = 0
        else
            accumulator = result
        end
    end
    return accumulator
end

-- pcall an object which is not a function (not callbacle)
-- should trigger this particular situation (see lj_err.c, function lj_err_optype_call)
-- Gross hack if lua_[p]call or pcall/xpcall fail for a non-callable object
-- L->base still points to the caller. So add a dummy frame with L instead
-- of a function. See lua_getstack().

local function pcall_wrong(x)
    if x % 2 == 0 then
        local ok, _ = pcall(123)
        if not ok then
            return 0
        end
    end
    local ok, result = pcall(hash_walk, x)
    if not ok then
        return 0
    end
    return result
end

local function test_pcall_not_a_function()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + pcall_wrong(i)
    end
    return accumulator
end

-- tail call multiple results (should trigger CALLMT byte code)

local function tail_call_multires(x)
    return hash_walk(x), x + 1, x + 2
end

-- a driver is needed, because tail-call elimination was prevented at the caller site

local function tail_multires_driver(x)
    return tail_call_multires(x)
end

local function test_tail_call_multires()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + tail_multires_driver(i)
    end
    return accumulator
end

-- pcall with multires

local function multires_maybe_error(x)
    if x % 5 == 0 then
        error("boom")
    end
    return hash_walk(x), x + 1, x + 2
end

local function pcall_multires(x)
    local ok, a, b, c = pcall(multires_maybe_error, x)
    if not ok then
        return 0
    end
    return a + b + c
end

local function test_pcall_multires()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + pcall_multires(i)
    end
    return accumulator
end

-- interpreter builtin function

local function builtin_c_call(x)
    return hash_walk(x + math.floor(math.sqrt(x + 1)))
end

local function test_builtin_c_call()
    local accumulator = 0
    for i = 0, ITER do
        accumulator = accumulator + builtin_c_call(i)
    end
    return accumulator
end

-- error during the argument evaluation

local function tail_call_multires_error(x)
    if x % 2 == 0 then
        return hash_walk(x), x + 1, error("boom")
    end
    return hash_walk(x), x + 1, x + 2
end

-- a driver is needed, because tail-call elimination was prevented at the caller site

local function tail_multires_error_driver(x)
    return tail_call_multires_error(x)
end

local function test_error_in_tail_call_multires_body(accumulator, i)
    return accumulator + tail_multires_error_driver(i)
end

local function test_error_in_tail_call_multires()
    local accumulator = 0
    for i = 0, ITER do
        ok, result = pcall(test_error_in_tail_call_multires_body, accumulator, i)
        if not ok then
            accumulator = 0
        else
            accumulator = result
        end
    end
    return accumulator
end


-- various tests for the profiler, uncomment one you want to execute

local tests = {
    --{ name = "plain_lua_call", fn = test_plain_lua_call },
    --{ name = "variadic", fn = test_variadic },
    --{ name = "pcall", fn = test_pcall },
    --{ name = "pcallh", fn = test_pcallh },
    --{ name = "pcallc", fn = test_pcallc },
    --{ name = "xpcall", fn = test_xpcall },
    --{ name = "pcall_error", fn = test_pcall_error },
    --{ name = "global", fn = test_global },
    --{ name = "tail_call", fn = test_tail_call },
    --{ name = "table_call", fn = test_table_call },
    --{ name = "table_method", fn = test_table_method },
    --{ name = "closure", fn = test_closure },
    --{ name = "indirect", fn = test_indirect },
    --{ name = "metatable_call", fn = test_metatable_call },
    --{ name = "metatable_index", fn = test_metatable_index },
    --{ name = "metatable_newindex", fn = test_metatable_newindex },
    --{ name = "metatable_string", fn = test_metatable_string },
    --{ name = "metatable_pairs", fn = test_metatable_pairs },
    --{ name = "coroutine", fn = test_coroutine },
    --{ name = "environment", fn = test_environment },
    --{ name = "module_call", fn = test_module_call },
    --{ name = "ffi_callback", fn = test_ffi_callback },
    --{ name = "gc", fn = test_gc },
    --{ name = "debug_hook", fn = test_debug_hook},
    --{ name = "error_in_ffi_callback", fn = test_error_in_ffi_callback },
    --{ name = "error_in_gc", fn = test_error_in_gc },
    --{ name = "error_in_debug_hook", fn = test_error_in_debug_hook },
    --{ name = "error_in_metatable_index", fn = test_error_in_metatable_index },
    --{ name = "pcall_not_a_function", fn = test_pcall_not_a_function },
    --{ name = "tail_call_multires", fn = test_tail_call_multires },
    --{ name = "pcall_multires", fn = test_pcall_multires },
    --{ name = "builtin_c_call", fn = test_builtin_c_call },
    { name = "error_in_tail_call_multires", fn = test_error_in_tail_call_multires },
}

local function run_with_jit(name, fn, jit_enabled)
    jit.flush()

    collectgarbage()
    collectgarbage()
    collectgarbage("stop")

    if jit_enabled then
        jit.on()
    else
        jit.off()
    end
    print(string.format("running %-20s | JIT %s", name, jit_enabled and "ON" or "OFF"))
    local result = fn()
    sink(result)
end

for _, test in ipairs(tests) do
    run_with_jit(test.name, test.fn, false) -- interpreter
    run_with_jit(test.name, test.fn, true) -- JIT
end
