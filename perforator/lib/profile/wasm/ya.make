PROGRAM(profile.wasm)

BUILD_ONLY_IF(OS_EMSCRIPTEN)

SRCS(
    profile.cpp
)

PEERDIR(
    perforator/lib/profile/c
)

LDFLAGS(
    -Wl,--strip-debug
    -Wl,--export=malloc
    -Wl,--export=free
    -Wl,--export=PerforatorProfileParsePProf
    -Wl,--export=PerforatorProfileDispose
    -Wl,--export=PerforatorWasmRender
    -Wl,--export=PerforatorStringData
    -Wl,--export=PerforatorStringSize
    -Wl,--export=PerforatorStringDispose
    -Wl,--export=PerforatorErrorString
    -Wl,--export=PerforatorErrorDispose
    -Wl,--export=__wasm_call_ctors
    -Wl,--export-memory
)

END()
