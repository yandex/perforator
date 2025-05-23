LIBRARY()

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm18/lib/Target/X86
)

PEERDIR(
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/Object
    contrib/libs/re2
    perforator/lib/elf
    perforator/lib/tls/parser
    perforator/lib/llvmex
    perforator/lib/php/asm/x86-64
)

SRCS(
    php.cpp
)

END()

RECURSE(
    asm
    cli
)
