PROGRAM(pythonparse)

ADDINCL(
    ${ARCADIA_BUILD_ROOT}/contrib/libs/llvm18/lib/Target/X86
)

SRCS(main.cpp)

PEERDIR(
    perforator/lib/python
    perforator/lib/llvmex
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/Object
)

END()
