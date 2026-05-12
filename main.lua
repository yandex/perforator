-- jit.off()

local ffi = require("ffi")
ffi.cdef [[
  typedef void (*cb)(void);
  void call(int n, void (*)(void));
  void loop(int n);
  void func(void);
  int rand(void);
]]
-- local callback = ffi.load("/home/spar/dev/perforator/callback.so")
local timeit = require("timeit")

local function lfunc() end

local UPVALUE = 0
local t = {}

function work()
  while true do
    -- print("C into C", timeit(function(n)
    --   callback.call(n, callback.func)
    -- end))

    -- print("Lua into C", timeit(function(n)
    --   for i = 1, n do callback.func() end
    -- end))

    -- print("C into Lua", timeit(function(n)
    --   callback.call(n, lfunc)
    -- end))

    print("Lua into Lua", timeit(function(n)
      for i = 1, n do lfunc() end
    end))

    -- print("C empty loop", timeit(function(n)
    --   callback.loop(n)
    -- end))

    print("Lua empty loop", timeit(function(n)
      for i = 1, n do UPVALUE = ffi.C.rand() t[n] = UPVALUE end
    end))
  end
end

work()

print("Done")
