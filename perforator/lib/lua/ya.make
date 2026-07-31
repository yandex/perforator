LIBRARY()

INCLUDE(${ARCADIA_ROOT}/perforator/lib/arch.ya.make.inc)

PEERDIR(
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/Object
    contrib/libs/re2
    perforator/lib/elf
    perforator/lib/tls/parser
    perforator/lib/llvmex
    perforator/lib/lua/asm
)

SRCS(
    lua.cpp
)

END()

RECURSE(
    asm
    cli
)
