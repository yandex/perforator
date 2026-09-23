PROGRAM(phpparse)

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm22/lib/Target/X86
)

SRCS(
    main.cpp
)

PEERDIR(
    contrib/libs/llvm22/include
    contrib/libs/llvm22/lib/Object
    perforator/lib/php
    perforator/lib/llvmex
)

END()
