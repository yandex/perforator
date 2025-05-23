PROGRAM(phpparse)

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm18/lib/Target/X86
)

SRCS(
    main.cpp
)

PEERDIR(
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/Object
    perforator/lib/php
    perforator/lib/llvmex
)

END()
