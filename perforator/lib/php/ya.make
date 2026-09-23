LIBRARY()

INCLUDE(${ARCADIA_ROOT}/perforator/lib/arch.ya.make.inc)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object
    contrib/libs/re2
    perforator/lib/elf
    perforator/lib/tls/parser
    perforator/lib/llvmex
    perforator/lib/php/asm
)

SRCS(
    php.cpp
)

END()

RECURSE(
    asm
    cli
)
